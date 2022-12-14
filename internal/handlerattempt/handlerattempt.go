package handlerattempt

import (
	"context"
	"fmt"
	"os"

	"github.com/hansmi/baamhackl/internal/config"
	"github.com/hansmi/baamhackl/internal/handlercommand"
	"github.com/hansmi/baamhackl/internal/journal"
	"github.com/hansmi/baamhackl/internal/waryio"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

func validateChangedFile(path string) (os.FileInfo, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file vanished before running command: %w", err)
		}

		return nil, err
	}

	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: not a regular file: %s", os.ErrInvalid, fi.Mode().Type())
	}

	return fi, nil
}

type Options struct {
	Logger  *zap.Logger
	Config  *config.Handler
	Journal *journal.Journal

	// Path to the changed file.
	ChangedFile string

	// Directory for storing execution-related files.
	BaseDir string

	// Whether the attempt is the last one before giving up.
	Final bool

	// Function to acquire a lock preventing concurrent file changes by handler
	// logic.
	AcquireLock func()
}

type Attempt struct {
	opts Options

	run func(context.Context) error
}

func New(opts Options) (*Attempt, error) {
	o := &Attempt{
		opts: opts,
	}

	if cmd, err := handlercommand.New(handlercommand.Options{
		Logger:     o.opts.Logger,
		SourceFile: o.opts.ChangedFile,
		BaseDir:    o.opts.BaseDir,
		Command:    o.opts.Config.Command,
	}); err != nil {
		return nil, err
	} else {
		o.run = cmd.Run
	}

	return o, nil
}

func (o *Attempt) moveToArchive(success bool) error {
	dest, err := o.opts.Journal.MoveToArchive(o.opts.ChangedFile, success)
	if err == nil && dest != "" {
		o.opts.Logger.Info("Moved changed file",
			zap.String("source", o.opts.ChangedFile),
			zap.String("dest", dest),
		)
	}

	return err
}

func (o *Attempt) Run(ctx context.Context) (bool, error) {
	statBefore, err := validateChangedFile(o.opts.ChangedFile)
	if err != nil {
		return true, err
	}

	o.opts.Logger.Info("File information",
		zap.String("name", statBefore.Name()),
		zap.Time("modtime", statBefore.ModTime()),
		zap.Int64("size", statBefore.Size()),
		zap.String("mode", statBefore.Mode().String()),
	)

	ctx, cancel := context.WithTimeout(ctx, o.opts.Config.Timeout)
	defer cancel()

	commandErr := o.run(ctx)

	if o.opts.AcquireLock != nil {
		o.opts.AcquireLock()
	}

	permanent := false
	combinedErr := commandErr

	// The changed file is moved if and only it still exists and remains
	// unchanged from before running the handler command.
	if statAfter, err := os.Lstat(o.opts.ChangedFile); err != nil {
		// Tolerate a missing file if and only if the command succeeded.
		if !(commandErr == nil && os.IsNotExist(err)) {
			multierr.AppendInto(&combinedErr, err)
		}

		// No point in retrying if the file doesn't exist anymore.
		permanent = os.IsNotExist(err)
	} else if changes := waryio.DescribeChanges(statBefore, statAfter); !changes.Empty() {
		multierr.AppendInto(&combinedErr, changes.Err())
	} else if o.opts.Final || commandErr == nil {
		multierr.AppendInto(&combinedErr, o.moveToArchive(commandErr == nil))
	}

	return permanent, combinedErr
}
