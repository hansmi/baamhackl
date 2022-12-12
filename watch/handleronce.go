package watch

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

type handlerOnce struct {
	cfg         *config.Handler
	journal     *journal.Journal
	changedFile string
	baseDir     string
	final       bool
	logger      *zap.Logger
	acquireLock func()

	run func(context.Context) error
}

func (o *handlerOnce) moveToArchive(success bool) error {
	src := o.changedFile
	dest, err := o.journal.MoveToArchive(src, success)
	if err == nil && dest != "" {
		o.logger.Info("Moved changed file", zap.String("source", src), zap.String("dest", dest))
	}

	return err
}

func (o *handlerOnce) do(ctx context.Context) (bool, error) {
	run := o.run

	if run == nil {
		c, err := handlercommand.New(handlercommand.Options{
			Logger:     o.logger,
			SourceFile: o.changedFile,
			BaseDir:    o.baseDir,
			Command:    o.cfg.Command,
		})
		if err != nil {
			return true, err
		}

		run = c.Run
	}

	statBefore, err := validateChangedFile(o.changedFile)
	if err != nil {
		return true, err
	}

	o.logger.Info("File information",
		zap.String("name", statBefore.Name()),
		zap.Time("modtime", statBefore.ModTime()),
		zap.Int64("size", statBefore.Size()),
		zap.String("mode", statBefore.Mode().String()),
	)

	ctx, cancel := context.WithTimeout(ctx, o.cfg.Timeout)
	defer cancel()

	commandErr := run(ctx)

	if o.acquireLock != nil {
		o.acquireLock()
	}

	permanent := false
	combinedErr := commandErr

	// The changed file is moved if and only it still exists and remains
	// unchanged from before running the handler command.
	if statAfter, err := os.Lstat(o.changedFile); err != nil {
		// Tolerate a missing file if and only if the command succeeded.
		if !(commandErr == nil && os.IsNotExist(err)) {
			multierr.AppendInto(&combinedErr, err)
		}

		// No point in retrying if the file doesn't exist anymore.
		permanent = os.IsNotExist(err)
	} else if changes := waryio.DescribeChanges(statBefore, statAfter); !changes.Empty() {
		multierr.AppendInto(&combinedErr, changes.Err())
	} else if o.final || commandErr == nil {
		multierr.AppendInto(&combinedErr, o.moveToArchive(commandErr == nil))
	}

	return permanent, combinedErr
}
