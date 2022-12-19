package journal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hansmi/baamhackl/internal/config"
	"github.com/hansmi/baamhackl/internal/prune"
	"github.com/hansmi/baamhackl/internal/uniquename"
	"github.com/hansmi/baamhackl/internal/waryio"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type dirOptions struct {
	path string
	uniquename.Options
}

type Journal struct {
	cfg *config.Handler

	journalDir dirOptions
	successDir dirOptions
	failureDir dirOptions
}

func New(cfg *config.Handler) *Journal {
	j := &Journal{
		cfg: cfg,
		journalDir: dirOptions{
			Options: uniquename.DefaultOptions,
			path:    cfg.JournalDir,
		},
		successDir: dirOptions{
			Options: uniquename.DefaultOptions,
			path:    cfg.SuccessDir,
		},
		failureDir: dirOptions{
			Options: uniquename.DefaultOptions,
			path:    cfg.FailureDir,
		},
	}

	j.journalDir.BeforeExtension = false

	return j
}

func (j *Journal) ensureDir(path string) (string, error) {
	return waryio.EnsureRelDir(j.cfg.Path, path, os.ModePerm)
}

func (j *Journal) ensureDirForName(d dirOptions, hint string) (*uniquename.Generator, error) {
	hint = filepath.Base(hint)

	if hint == "" {
		return nil, fmt.Errorf("%w: non-empty hint is required", os.ErrInvalid)
	}

	base, err := j.ensureDir(d.path)
	if err != nil {
		return nil, err
	}

	return uniquename.New(filepath.Join(base, hint), d.Options)
}

func (j *Journal) CreateTaskDir(hint string) (string, error) {
	g, err := j.ensureDirForName(j.journalDir, hint)
	if err != nil {
		return "", err
	}

	return waryio.MakeAvailableDir(g)
}

func (j *Journal) MoveToArchive(path string, success bool) (string, error) {
	destDir := j.failureDir

	if success {
		destDir = j.successDir
	}

	g, err := j.ensureDirForName(destDir, filepath.Base(path))
	if err != nil {
		return "", err
	}

	return waryio.RenameToAvailableName(path, g)
}

func (j *Journal) Prune(ctx context.Context, logger *zap.Logger) error {
	deadline := time.Now().Add(-j.cfg.JournalRetention).Truncate(time.Minute)

	all := []dirOptions{
		j.journalDir,
		j.successDir,
		j.failureDir,
	}

	var pruners []prune.Pruner
	var allPaths []string

	for _, i := range all {
		dir, err := j.ensureDir(i.path)
		if err != nil {
			return err
		}

		pruners = append(pruners, prune.Pruner{
			Dir:    dir,
			Accept: prune.MakeAgeFilter(deadline, i.Options),
			Logger: logger.With(zap.String("dir", dir)),
		})

		allPaths = append(allPaths, dir)
	}

	logger.Info("Pruning journal",
		zap.Time("deadline", deadline),
		zap.Strings("dirs", allPaths))

	var allErrors error

	for _, i := range pruners {
		multierr.AppendInto(&allErrors, i.Run(ctx))
	}

	return allErrors
}
