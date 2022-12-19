package watch

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hansmi/baamhackl/internal/config"
	"github.com/hansmi/baamhackl/internal/handlertask"
	"github.com/hansmi/baamhackl/internal/scheduler"
	"github.com/hansmi/baamhackl/internal/service"
	"github.com/hansmi/baamhackl/internal/testutil"
)

var errTest = errors.New("test error")

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
				h.invoke = func(ctx context.Context, t *handlertask.Task, acquireLock func()) error {
					if lockEarly {
						acquireLock()
					}

					return tc.err
				}

				ctx, cancel := context.WithCancel(context.Background())
				t.Cleanup(cancel)

				err := h.invokeTask(ctx, handlertask.New(handlertask.Options{
					Name: "test",
				}))

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

		metricNames []string
		wantMetrics string
	}{
		{
			name: "success",
			metricNames: []string{
				"retries_total",
				"failures_total",
				"finished_total",
				"pending_total",
			},
			wantMetrics: `
			# HELP retries_total Number of retries.
			# TYPE retries_total counter
			retries_total 0
			# HELP failures_total Number of failures.
			# TYPE failures_total counter
			failures_total 0
			# HELP finished_total Total number of handled changes (including failures).
			# TYPE finished_total counter
			finished_total 1
			# HELP pending_total Number of currently waiting tasks.
			# TYPE pending_total gauge
			pending_total 0
			`,
		},
		{
			name:     "success after retries",
			attempts: 10,
			metricNames: []string{
				"retries_total",
				"failures_total",
				"finished_total",
				"pending_total",
			},
			wantMetrics: `
			# HELP retries_total Number of retries.
			# TYPE retries_total counter
			retries_total 10
			# HELP failures_total Number of failures.
			# TYPE failures_total counter
			failures_total 0
			# HELP finished_total Total number of handled changes (including failures).
			# TYPE finished_total counter
			finished_total 1
			# HELP pending_total Number of currently waiting tasks.
			# TYPE pending_total gauge
			pending_total 0
			`,
		},
		{
			name:    "failure",
			err:     errTest,
			wantErr: errTest,
			metricNames: []string{
				"retries_total",
				"failures_total",
				"finished_total",
				"pending_total",
			},
			wantMetrics: `
			# HELP retries_total Number of retries.
			# TYPE retries_total counter
			retries_total 0
			# HELP failures_total Number of failures.
			# TYPE failures_total counter
			failures_total 1
			# HELP finished_total Total number of handled changes (including failures).
			# TYPE finished_total counter
			finished_total 1
			# HELP pending_total Number of currently waiting tasks.
			# TYPE pending_total gauge
			pending_total 0
			`,
		},
		{
			name:     "failure after retries",
			attempts: 10,
			err:      errTest,
			wantErr:  errTest,
			metricNames: []string{
				"retries_total",
				"failures_total",
				"finished_total",
			},
			wantMetrics: `
			# HELP retries_total Number of retries.
			# TYPE retries_total counter
			retries_total 10
			# HELP failures_total Number of failures.
			# TYPE failures_total counter
			failures_total 1
			# HELP finished_total Total number of handled changes (including failures).
			# TYPE finished_total counter
			finished_total 1
			`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.HandlerDefaults
			cfg.RetryCount = 0
			cfg.RetryDelayMax = 0
			cfg.Path = rootDir

			calls := 0

			h := newHandler(&cfg)
			h.invoke = func(ctx context.Context, task *handlertask.Task, acquireLock func()) error {
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

			if diff := cmp.Diff(map[string]*handlertask.Task{}, h.pending); diff != "" {
				t.Errorf("Pending tasks diff (-want +got):\n%s", diff)
			}
			h.mu.Unlock()

			testutil.CollectAndCompare(t, h.metrics(), tc.wantMetrics, tc.metricNames...)
		})
	}
}
