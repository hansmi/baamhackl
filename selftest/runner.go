package selftest

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/goccy/go-yaml"
	"github.com/hansmi/baamhackl/internal/config"
	"github.com/hansmi/baamhackl/internal/waryio"
	"github.com/hansmi/baamhackl/internal/watchman"
	"github.com/hansmi/baamhackl/selftest/commandenv"
	"github.com/hansmi/baamhackl/selftest/multifile"
	"github.com/hansmi/baamhackl/watch"
	"go.uber.org/multierr"
	"golang.org/x/sync/errgroup"
)

type test interface {
	Name() string
	HandlerConfig() map[string]any
	Setup() error
	Run(context.Context) error
}

func writeConfig(dir string, data any) (string, error) {
	fh, err := os.CreateTemp(dir, "config*")
	if err != nil {
		return "", err
	}

	if buf, err := yaml.Marshal(data); err != nil {
		return "", err
	} else if _, err = fh.Write(buf); err != nil {
		return "", err
	}

	if err := fh.Close(); err != nil {
		return "", err
	}

	return fh.Name(), nil
}

type runner struct {
	baseDir  string
	inputDir string
	tests    []test
}

func newRunner(dir string) (*runner, error) {
	var err error

	r := &runner{
		baseDir: dir,
	}

	if r.inputDir, err = waryio.EnsureRelDir(r.baseDir, "input", os.ModePerm); err != nil {
		return nil, err
	}

	if t, err := commandenv.New(r.baseDir); err != nil {
		return nil, err
	} else {
		r.tests = append(r.tests, t)
	}

	if t, err := multifile.New(r.baseDir); err != nil {
		return nil, err
	} else {
		r.tests = append(r.tests, t)
	}

	return r, nil
}

func (r *runner) generateConfig() any {
	var handlers []any

	for _, t := range r.tests {
		handlers = append(handlers, t.HandlerConfig())
	}

	return map[string]any{
		"handlers": handlers,
	}
}

// setupAll configures all tests.
func (r *runner) setupAll() error {
	var allErrors error

	for _, t := range r.tests {
		if err := t.Setup(); err != nil {
			multierr.AppendInto(&allErrors, fmt.Errorf("test %q: %w", t.Name(), err))
		}
	}

	return allErrors
}

func (r *runner) newWatchCommand(configFile string) (*watch.Command, error) {
	args := []string{
		"-config", configFile,
	}

	var watchCmd watch.Command
	var fs flag.FlagSet

	watchCmd.SetFlags(&fs)

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	return &watchCmd, nil
}

// runTest executes all tests and waits for them to finish.
func (r *runner) runAll(ctx context.Context) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var allErrors error

	for _, t := range r.tests {
		wg.Add(1)
		go func(t test) {
			defer wg.Done()

			if err := t.Run(ctx); err != nil {
				err = fmt.Errorf("test %q: %w", t.Name(), err)

				mu.Lock()
				multierr.AppendInto(&allErrors, err)
				mu.Unlock()
			}
		}(t)
	}

	wg.Wait()

	return allErrors
}

func (r *runner) run(ctx context.Context, client watchman.Client) error {
	configFile, err := writeConfig(r.baseDir, r.generateConfig())
	if err != nil {
		return err
	}

	if err := os.Unsetenv(config.PathEnvVar); err != nil {
		return err
	}

	watchCmd, err := r.newWatchCommand(configFile)
	if err != nil {
		return err
	}

	if err := r.setupAll(); err != nil {
		return err
	}

	eg, egCtx := errgroup.WithContext(ctx)

	watchCtx, watchCancel := context.WithCancel(egCtx)
	defer watchCancel()

	testCtx, testCancel := context.WithCancel(egCtx)
	defer testCancel()

	eg.Go(func() error {
		defer testCancel()

		if err := watchCmd.ExecuteWithClient(watchCtx, client); !(err == nil || errors.Is(err, context.Canceled)) {
			return err
		}

		return nil
	})

	eg.Go(func() error {
		defer watchCancel()

		return r.runAll(testCtx)
	})

	return eg.Wait()
}
