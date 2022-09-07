package prune

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
	"github.com/hansmi/baamhackl/internal/uniquename"
	"github.com/spf13/afero"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

const lockName = ".prune.lock"

var ErrUnavailable = errors.New("lock unavailable")

type AcceptFunc func(string, os.FileInfo) bool

// MakeAgeFilter returns an acceptor function only permitting files older than
// deadline. Both the file's modification time and, if available, the time in
// the filename must be older.
func MakeAgeFilter(deadline time.Time, opts uniquename.Options) AcceptFunc {
	return func(name string, fi os.FileInfo) bool {
		if deadline.Before(fi.ModTime()) {
			return false
		}

		if ts, err := uniquename.ExtractTime(name, opts); err == nil {
			return !ts.IsZero() && ts.Before(deadline)
		}

		return true
	}
}

// Pruner removes files and directories accepted by the filter function.
type Pruner struct {
	Logger *zap.Logger
	Dir    string
	Accept AcceptFunc

	fs afero.Fs
}

func (p Pruner) runLocked(ctx context.Context) (resultErr error) {
	logger := p.Logger
	fs := p.fs

	if logger == nil {
		logger = zap.NewNop()
	}

	if fs == nil {
		fs = afero.NewOsFs()
	}

	entries, err := afero.ReadDir(fs, p.Dir)
	if err != nil {
		return err
	}

loop:
	for _, fi := range entries {
		select {
		case <-ctx.Done():
			multierr.AppendInto(&resultErr, ctx.Err())
			break loop
		default:
		}

		if fi.Name() != lockName && p.Accept(fi.Name(), fi) {
			path := filepath.Join(p.Dir, fi.Name())

			logger.Info(fmt.Sprintf("Removing entry %q", fi.Name()),
				zap.Time("modified", fi.ModTime()))

			if err := fs.RemoveAll(path); !(err == nil || os.IsNotExist(err)) {
				multierr.AppendInto(&resultErr, err)
			}
		}
	}

	return resultErr
}

func (p Pruner) Run(ctx context.Context) (resultErr error) {
	lock := flock.New(filepath.Join(p.Dir, lockName))

	if locked, err := lock.TryLock(); err != nil {
		return err
	} else if !locked {
		return ErrUnavailable
	}

	defer multierr.AppendInvoke(&resultErr, multierr.Close(lock))

	return p.runLocked(ctx)
}
