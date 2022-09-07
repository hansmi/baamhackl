package watch

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hansmi/baamhackl/internal/config"
	"github.com/hansmi/baamhackl/internal/scheduler"
	"github.com/hansmi/baamhackl/internal/service"
)

func TestHandlerInvokeTask(t *testing.T) {
	for _, tc := range []struct {
		name    string
		err     error
		wantErr error
	}{
		{
			name: "success",
		},
		{
			name:    "failure",
			err:     errTest,
			wantErr: errTest,
		},
		{
			name:    "failure as TaskError",
			err:     scheduler.AsTaskError(errTest),
			wantErr: errTest,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.HandlerDefaults
			cfg.Path = t.TempDir()

			for _, lockEarly := range []bool{false, true} {
				h := newHandler(&cfg)
				h.invoke = func(ctx context.Context, t *handlerTask, acquireLock func()) error {
					if lockEarly {
						acquireLock()
					}

					return tc.err
				}

				ctx, cancel := context.WithCancel(context.Background())
				t.Cleanup(cancel)

				err := h.invokeTask(ctx, &handlerTask{
					name: "test",
				})

				if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
					t.Errorf("Error diff (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestHandler(t *testing.T) {
	sched := scheduler.New()
	sched.Start()

	t.Cleanup(func() {
		if err := sched.Stop(context.Background()); err != nil {
			t.Errorf("Stop() failed: %v", err)
		}
	})

	rootDir := t.TempDir()

	for _, tc := range []struct {
		name     string
		attempts int
		err      error
		wantErr  error
	}{
		{
			name: "success",
		},
		{
			name:     "success after retries",
			attempts: 10,
		},
		{
			name:    "failure",
			err:     errTest,
			wantErr: errTest,
		},
		{
			name:     "failure after retries",
			attempts: 10,
			err:      errTest,
			wantErr:  errTest,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.HandlerDefaults
			cfg.RetryCount = 0
			cfg.RetryDelayMax = 0
			cfg.Path = rootDir

			calls := 0

			h := newHandler(&cfg)
			h.invoke = func(ctx context.Context, task *handlerTask, acquireLock func()) error {
				acquireLock()

				if calls++; calls <= tc.attempts {
					return &scheduler.TaskError{
						Err:        tc.err,
						RetryDelay: time.Nanosecond,
					}
				}

				return tc.err
			}

			req := service.FileChangedRequest{
				RootDir: rootDir,
			}
			req.Change.Name = filepath.Join("dir", tc.name)

			if err := h.handle(sched, req); err != nil {
				t.Errorf("handle(%+v) failed: %v", req, err)
			}

			if err := sched.Quiesce(context.Background()); err != nil {
				t.Errorf("Quiesce() failed: %v", err)
			}

			h.mu.Lock()
			if diff := cmp.Diff(tc.attempts+1, calls); diff != "" {
				t.Errorf("Attempt count diff (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(map[string]*handlerTask{}, h.pending); diff != "" {
				t.Errorf("Pending tasks diff (-want +got):\n%s", diff)
			}
			h.mu.Unlock()
		})
	}
}
