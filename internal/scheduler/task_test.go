package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hansmi/baamhackl/internal/testutil"
	"github.com/jonboulle/clockwork"
)

var errTest = errors.New("test failure")

func runForTest(t *testing.T, task *Task, wantFinished bool) *Task {
	t.Helper()

	// defer zap.ReplaceGlobals(zaptest.NewLogger(t))()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if finished := task.run(ctx); finished != wantFinished {
		t.Errorf("run() returned %t, want %t", finished, wantFinished)
	}

	return task
}

func TestTaskRun(t *testing.T) {
	fc := clockwork.NewFakeClock()

	testutil.ReplaceClock(t, &clock, fc)

	for _, tc := range []struct {
		name         string
		err          error
		wantFinished bool
		wantAfter    time.Time
	}{
		{
			name:         "success",
			wantFinished: true,
		},
		{
			name: "no backoff",
			err: &TaskError{
				Err:        errTest,
				RetryDelay: 0,
			},
			wantAfter: fc.Now(),
		},
		{
			name: "one minute",
			err: &TaskError{
				Err:        errTest,
				RetryDelay: time.Minute,
			},
			wantAfter: fc.Now().Add(time.Minute),
		},
		{
			name: "permanent",
			err: &TaskError{
				Err:        errTest,
				RetryDelay: Stop,
			},
			wantFinished: true,
		},
		{
			name:         "permanent, plain error",
			err:          errTest,
			wantFinished: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			task := runForTest(t, &Task{
				fn: func(context.Context) error {
					return tc.err
				},
			}, tc.wantFinished)

			if diff := cmp.Diff(tc.wantAfter, task.nextAfter, cmpopts.EquateApproxTime(time.Millisecond)); diff != "" {
				t.Errorf("NextAfter diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTaskRunMultipleCalls(t *testing.T) {
	fc := clockwork.NewFakeClock()

	testutil.ReplaceClock(t, &clock, fc)

	count := 0
	task := &Task{
		fn: func(context.Context) error {
			defer func() { count++ }()

			if count < 3 {
				// Temporary error
				return &TaskError{
					Err:        errTest,
					RetryDelay: time.Duration(count) * time.Millisecond,
				}
			}

			// Permanent error
			return errTest
		},
	}

	for i := 0; i < 10; i++ {
		done := (i == 3)

		task = runForTest(t, task, done)

		if done {
			break
		}
	}
}
