package watch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/hansmi/baamhackl/internal/config"
	"github.com/hansmi/baamhackl/internal/scheduler"
	"github.com/hansmi/baamhackl/internal/teelog"
	"github.com/hansmi/baamhackl/internal/waryio"
	"go.uber.org/zap"
)

type handlerTask struct {
	cfg     *config.Handler
	journal *journal

	// Name of modified file
	name string

	retry          *handlerRetryStrategy
	currentAttempt int
	journalDir     string
	fuzzFactor     float32

	invoke func(context.Context, *handlerOnce) (bool, error)
}

func (t *handlerTask) ensureJournalDir() error {
	if t.journalDir == "" {
		path, err := t.journal.createTaskDir(t.name)
		if err != nil {
			return fmt.Errorf("creating journal directory failed: %w", err)
		}
		t.journalDir = path
	}

	return nil
}

func (t *handlerTask) run(ctx context.Context, acquireLock func()) error {
	logger := zap.L().With(
		zap.String("root", t.cfg.Path),
		zap.String("name", t.name),
	)
	logger.Info("Handling changed file", zap.Int("attempt", t.currentAttempt))

	defer func() {
		t.currentAttempt++
	}()

	if err := t.ensureJournalDir(); err != nil {
		return err
	}

	if t.retry == nil {
		t.retry = newHandlerRetryStrategy(*t.cfg)
	}

	taskLogger := teelog.File{
		Parent: logger,

		// The same log file is used for all attempts.
		Path: filepath.Join(t.journalDir, "log.txt"),
	}

	retryDelay := t.retry.current()
	var permanent bool

	err := taskLogger.Wrap(func(inner *zap.Logger) error {
		taskDir, err := waryio.EnsureRelDir(t.journalDir, strconv.FormatInt(int64(t.currentAttempt), 10), os.ModePerm)
		if err != nil {
			return err
		}

		if t.invoke == nil {
			t.invoke = func(ctx context.Context, o *handlerOnce) (bool, error) {
				return o.do(ctx)
			}
		}

		permanent, err = t.invoke(ctx, &handlerOnce{
			cfg:         t.cfg,
			changedFile: filepath.Join(t.cfg.Path, t.name),
			baseDir:     taskDir,

			// Is this the last attempt?
			final: retryDelay == scheduler.Stop,

			logger:      inner,
			acquireLock: acquireLock,
		})

		if err != nil {
			inner.Error("Handling file change failed", zap.Error(err))
		}

		return err
	})

	if permanent || err == nil {
		return err
	}

	t.retry.advance()

	return &scheduler.TaskError{
		Err:        err,
		RetryDelay: fuzzDuration(retryDelay, t.fuzzFactor),
	}
}
