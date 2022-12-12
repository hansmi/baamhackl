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

var ArchiveNamingOptions = uniquename.DefaultOptions

type Journal struct {
	cfg            *config.Handler
	taskDirOptions uniquename.Options
}

func New(cfg *config.Handler) *Journal {
	j := &Journal{
		cfg:            cfg,
		taskDirOptions: uniquename.DefaultOptions,
	}
	j.taskDirOptions.BeforeExtension = false
	return j
}

func (j *Journal) ensureDir(path string) (string, error) {
	return waryio.EnsureRelDir(j.cfg.Path, path, os.ModePerm)
}

func (j *Journal) CreateTaskDir(hint string) (string, error) {
	if hint == "" {
		return "", fmt.Errorf("%w: non-empty hint is required", os.ErrInvalid)
	}

	base, err := j.ensureDir(j.cfg.JournalDir)
	if err != nil {
		return "", err
	}

	g, err := uniquename.New(filepath.Join(base, filepath.Base(hint)), j.taskDirOptions)
	if err != nil {
		return "", err
	}

	return waryio.MakeAvailableDir(g)
}

func (j *Journal) Prune(ctx context.Context, logger *zap.Logger) error {
	deadline := time.Now().Add(-j.cfg.JournalRetention).Truncate(time.Minute)

	type info struct {
		path string
		opts uniquename.Options
	}

	all := []*info{
		{j.cfg.JournalDir, j.taskDirOptions},
		{j.cfg.SuccessDir, ArchiveNamingOptions},
		{j.cfg.FailureDir, ArchiveNamingOptions},
	}

	var dirNames []string

	for _, i := range all {
		path, err := j.ensureDir(i.path)
		if err != nil {
			return err
		}
		i.path = path

		dirNames = append(dirNames, path)
	}

	logger.Info("Pruning journal",
		zap.Time("deadline", deadline),
		zap.Strings("dirs", dirNames))

	var allErrors error

	for _, i := range all {
		multierr.AppendInto(&allErrors, prune.Pruner{
			Dir:    i.path,
			Accept: prune.MakeAgeFilter(deadline, i.opts),
			Logger: logger.With(zap.String("dir", i.path)),
		}.Run(ctx))
	}

	return allErrors
}
