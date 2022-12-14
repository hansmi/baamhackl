package watch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/hansmi/baamhackl/internal/config"
	"github.com/hansmi/baamhackl/internal/handlerattempt"
	"github.com/hansmi/baamhackl/internal/handlerretrystrategy"
	"github.com/hansmi/baamhackl/internal/journal"
	"github.com/hansmi/baamhackl/internal/scheduler"
	"github.com/hansmi/baamhackl/internal/teelog"
	"github.com/hansmi/baamhackl/internal/waryio"
	"go.uber.org/zap"
)

type handlerTask struct {
	cfg     *config.Handler
	journal *journal.Journal

	// Name of modified file
	name string

	retry          *handlerretrystrategy.Strategy
	currentAttempt int
	journalDir     string
	fuzzFactor     float32

	invoke func(context.Context, handlerattempt.Options) (bool, error)
}

func (t *handlerTask) ensureJournalDir() error {
	if t.journalDir == "" {
		path, err := t.journal.CreateTaskDir(t.name)
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
		t.retry = handlerretrystrategy.New(*t.cfg)
	}

	taskLogger := teelog.File{
		Parent: logger,

		// The same log file is used for all attempts.
		Path: filepath.Join(t.journalDir, "log.txt"),
	}

	retryDelay := t.retry.Current()
	var permanent bool

	err := taskLogger.Wrap(func(inner *zap.Logger) error {
		taskDir, err := waryio.EnsureRelDir(t.journalDir, strconv.FormatInt(int64(t.currentAttempt), 10), os.ModePerm)
		if err != nil {
			return err
		}

		if t.invoke == nil {
			t.invoke = func(ctx context.Context, opts handlerattempt.Options) (bool, error) {
				h, err := handlerattempt.New(opts)
				if err != nil {
					return true, err
				}

				return h.Run(ctx)
			}
		}

		permanent, err = t.invoke(ctx, handlerattempt.Options{
			Logger: inner,

			Config:      t.cfg,
			Journal:     t.journal,
			ChangedFile: filepath.Join(t.cfg.Path, t.name),
			BaseDir:     taskDir,

			// Is this the last attempt?
			Final: retryDelay == scheduler.Stop,

			AcquireLock: acquireLock,
		})
		if err != nil {
			inner.Error("Handling file change failed", zap.Error(err))
		}

		return err
	})

	if permanent || err == nil {
		return err
	}

	t.retry.Advance()

	return &scheduler.TaskError{
		Err:        err,
		RetryDelay: fuzzDuration(retryDelay, t.fuzzFactor),
	}
}
