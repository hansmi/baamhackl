package scheduler

import (
	"context"
	"time"

	"github.com/hansmi/baamhackl/internal/prioqueue"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// TaskFunc is the signature of functions implementing tasks. They're called
// when it's the task's turn. Returning nil indicates success and the task is
// removed from the queue. *TaskError can be returned to configure a retry
// delay. A non-nil error of another type is always a permanent failure.
type TaskFunc func(context.Context) error

// Task describes one scheduled tasks.
type Task struct {
	fn func(context.Context) error

	// Unique sequence number
	seq int64

	// Execute task only after this point in time
	nextAfter time.Time

	// Number of the next execution attempt
	attemptCount int

	seqItem  *prioqueue.Item
	timeItem *prioqueue.Item
}

type taskLogAdapter struct {
	*Task
}

func (a taskLogAdapter) MarshalLogObject(oe zapcore.ObjectEncoder) error {
	oe.AddInt64("seq", a.seq)
	return nil
}

// Execute the task and return whether it's finished.
func (t *Task) run(ctx context.Context) bool {
	logFields := []zap.Field{
		zap.Object("task", taskLogAdapter{t}),
		zap.Int("attempt", t.attemptCount),
	}

	t.attemptCount++

	logger := zap.L()
	logger.Info("Starting task", logFields...)

	if err := t.fn(ctx); err == nil {
		logger.Info("Task successful", logFields...)
	} else {
		logFields = append(logFields, zap.Error(err))

		if te := AsTaskError(err); te.Permanent() {
			logger.Error("Task failed permanently", logFields...)
		} else {
			t.nextAfter = clock.Now().Add(te.RetryDelay)
			logFields = append(logFields,
				zap.Duration("retry_delay", te.RetryDelay),
				zap.Time("retry_after", t.nextAfter),
			)
			logger.Error("Task failed and will be attempted again", logFields...)
			return false
		}
	}

	return true
}
