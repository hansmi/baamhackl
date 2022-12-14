package watch

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hansmi/baamhackl/internal/config"
	"github.com/hansmi/baamhackl/internal/handlerattempt"
	"github.com/hansmi/baamhackl/internal/journal"
	"github.com/hansmi/baamhackl/internal/scheduler"
	"github.com/hansmi/baamhackl/internal/testutil"
)

func TestHandlerTaskEnsureJournalDir(t *testing.T) {
	cfg := config.HandlerDefaults
	cfg.Path = t.TempDir()

	task := &handlerTask{
		cfg:     &cfg,
		journal: journal.New(&cfg),
		name:    "test.txt",
	}

	baseJournalDir := filepath.Join(cfg.Path, cfg.JournalDir)
	testutil.MustNotExist(t, baseJournalDir)

	for i := 0; i < 3; i++ {
		err := task.ensureJournalDir()

		if diff := cmp.Diff(nil, err, cmpopts.EquateErrors()); diff != "" {
			t.Errorf("Error diff (-want +got):\n%s", diff)
		}

		testutil.MustLstat(t, baseJournalDir)

		if st := testutil.MustLstat(t, task.journalDir); !st.IsDir() {
			t.Errorf("Not a directory: %+v", st)
		}
	}
}

func TestHandlerTaskRun(t *testing.T) {
	for _, tc := range []struct {
		name       string
		retryCount int
		invoke     func() (bool, error)
		wantErr    error
		wantDelay  time.Duration
	}{
		{
			name: "permanent success",
			invoke: func() (bool, error) {
				return true, nil
			},
		},
		{
			name: "temporary success",
			invoke: func() (bool, error) {
				return false, nil
			},
		},
		{
			name: "permanent failure",
			invoke: func() (bool, error) {
				return true, errTest
			},
			wantErr:   errTest,
			wantDelay: scheduler.Stop,
		},
		{
			name: "temporary failure without retries",
			invoke: func() (bool, error) {
				return false, errTest
			},
			wantErr:   errTest,
			wantDelay: scheduler.Stop,
		},
		{
			name:       "temporary failure with retries",
			retryCount: 5,
			invoke: func() (bool, error) {
				return false, errTest
			},
			wantErr:   errTest,
			wantDelay: config.HandlerDefaults.RetryDelayInitial,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.HandlerDefaults
			cfg.RetryCount = tc.retryCount
			cfg.RetryDelayFactor = 1
			cfg.Path = t.TempDir()

			testutil.MustWriteFile(t, filepath.Join(cfg.Path, "test.txt"), "content")

			task := &handlerTask{
				cfg:     &cfg,
				journal: journal.New(&cfg),
				name:    "test.txt",
				invoke: func(ctx context.Context, opts handlerattempt.Options) (bool, error) {
					testutil.MustLstat(t, opts.ChangedFile)

					return tc.invoke()
				},
			}

			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			for done, attempt := false, 0; !done; attempt++ {
				err := task.run(ctx, nil)

				if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
					t.Errorf("Error diff (-want +got):\n%s", diff)
				}

				delay := scheduler.Stop

				if err != nil {
					delay = scheduler.AsTaskError(err).RetryDelay
				}

				wantDelay := tc.wantDelay

				if attempt == tc.retryCount {
					wantDelay = scheduler.Stop
					done = true
				}

				if diff := cmp.Diff(wantDelay, delay); diff != "" {
					t.Errorf("Task delay diff (-want +got):\n%s", diff)
				}

				testutil.MustLstat(t, filepath.Join(task.journalDir, "log.txt"))
				if st := testutil.MustLstat(t, filepath.Join(task.journalDir, fmt.Sprint(attempt))); !st.IsDir() {
					t.Errorf("Not a directory: %+v", st)
				}
				testutil.MustNotExist(t, filepath.Join(task.journalDir, fmt.Sprint(attempt+1)))
			}
		})
	}
}
