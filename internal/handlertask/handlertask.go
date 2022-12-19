package handlertask

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/hansmi/baamhackl/internal/config"
	"github.com/hansmi/baamhackl/internal/fuzzduration"
	"github.com/hansmi/baamhackl/internal/handlerattempt"
	"github.com/hansmi/baamhackl/internal/handlerretrystrategy"
	"github.com/hansmi/baamhackl/internal/journal"
	"github.com/hansmi/baamhackl/internal/scheduler"
	"github.com/hansmi/baamhackl/internal/teelog"
	"github.com/hansmi/baamhackl/internal/waryio"
	"go.uber.org/zap"
)

type MetricsReporter interface {
	handlerattempt.MetricsReporter
}

type Options struct {
	Config  *config.Handler
	Journal *journal.Journal

	// Name of modified file
	Name string

	// Interface for reporting metrics.
	Metrics MetricsReporter
}

type Task struct {
	opts Options

	retry          *handlerretrystrategy.Strategy
	currentAttempt int
	journalDir     string
	fuzzFactor     float32

	invoke func(context.Context, handlerattempt.Options) (bool, error)
}

func New(opts Options) *Task {
	return &Task{
		opts:       opts,
		fuzzFactor: 0.1,
	}
}

func (t *Task) Name() string {
	return t.opts.Name
}

func (t *Task) ensureJournalDir() error {
	if t.journalDir == "" {
		path, err := t.opts.Journal.CreateTaskDir(t.opts.Name)
		if err != nil {
			return fmt.Errorf("creating journal directory failed: %w", err)
		}
		t.journalDir = path
	}

	return nil
}

func (t *Task) Run(ctx context.Context, acquireLock func()) error {
	logger := zap.L().With(
		zap.String("root", t.opts.Config.Path),
		zap.String("name", t.opts.Name),
	)
	logger.Info("Handling changed file", zap.Int("attempt", t.currentAttempt))

	defer func() {
		t.currentAttempt++
	}()

	if err := t.ensureJournalDir(); err != nil {
		return err
	}

	if t.retry == nil {
		t.retry = handlerretrystrategy.New(*t.opts.Config)
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

	taskLogger := teelog.File{
		Parent: logger,

		// The same log file is used for all attempts.
		Path: filepath.Join(t.journalDir, "log.txt"),
	}

	retryDelay := t.retry.Current()
	var permanent bool

	err := taskLogger.Wrap(func(inner *zap.Logger) error {
		taskDir, err := waryio.EnsureRelDir(t.journalDir, strconv.Itoa(t.currentAttempt), os.ModePerm)
		if err != nil {
			return err
		}

		permanent, err = t.invoke(ctx, handlerattempt.Options{
			Logger:  inner,
			Metrics: t.opts.Metrics,

			Config:      t.opts.Config,
			Journal:     t.opts.Journal,
			ChangedFile: filepath.Join(t.opts.Config.Path, t.opts.Name),
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
		RetryDelay: fuzzduration.Random(retryDelay, t.fuzzFactor),
	}
}
