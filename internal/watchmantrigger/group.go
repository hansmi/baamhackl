package watchmantrigger

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/hansmi/baamhackl/internal/config"
	"github.com/hansmi/baamhackl/internal/exepath"
	"github.com/hansmi/baamhackl/internal/watchman"
	"go.uber.org/multierr"
	"golang.org/x/sync/errgroup"
)

const maxConcurrent = 3
const configFileLocalScope = ".watchmanconfig"
const configFileLocalScopeMode = 0o600

type triggerSetter struct {
	client  watchman.Client
	command []string
}

func newTriggerSetter(client watchman.Client, socketPath string) (*triggerSetter, error) {
	exe, err := exepath.Get()
	if err != nil {
		return nil, err
	}

	return &triggerSetter{
		client:  client,
		command: []string{exe, "send-file-changes", socketPath},
	}, nil
}

func (s *triggerSetter) do(ctx context.Context, h *config.Handler) error {
	cfg, err := newTriggerConfig(*h)
	if err != nil {
		return err
	}

	if configContent, err := cfg.configDataJSON(); err != nil {
		return err
	} else if err := os.WriteFile(cfg.configFilePath, configContent, configFileLocalScopeMode); err != nil {
		return err
	} else if err := os.Chmod(cfg.configFilePath, configFileLocalScopeMode); err != nil {
		return err
	}

	if err := s.client.WatchSet(ctx, h.Path); err != nil {
		return err
	}

	args := map[string]any{
		"name":       h.Name,
		"expression": cfg.expression,
		"stdin":      watchman.FileChangeFields,
		"command":    s.command,
	}

	if err := s.client.TriggerSet(ctx, h.Path, args); err != nil {
		return fmt.Errorf("trigger %q: %w", h.Name, err)
	}

	return nil
}

// Group is a collection of triggers configured in Watchman. Calling DeleteAll
// unconfigures them.
type Group struct {
	Client     watchman.Client
	SocketPath string

	mu         sync.Mutex
	configured []*config.Handler
}

// SetAll configures triggers for all given handlers.
func (g *Group) SetAll(ctx context.Context, all []*config.Handler) error {
	ts, err := newTriggerSetter(g.Client, g.SocketPath)
	if err != nil {
		return err
	}

	eg, gctx := errgroup.WithContext(ctx)
	eg.SetLimit(maxConcurrent)

	for _, h := range all {
		h := h

		eg.Go(func() error {
			if err := ts.do(gctx, h); err != nil {
				return err
			}

			g.mu.Lock()
			g.configured = append(g.configured, h)
			g.mu.Unlock()

			return nil
		})
	}

	return eg.Wait()
}

// RecrawlAll triggers a full recrawl on all watched directories.
func (g *Group) RecrawlAll(ctx context.Context) error {
	eg, gctx := errgroup.WithContext(ctx)
	eg.SetLimit(maxConcurrent)

	for _, h := range g.configured {
		h := h

		eg.Go(func() error {
			return g.Client.Recrawl(gctx, h.Path)
		})
	}

	return eg.Wait()
}

// DeleteAll unconfigures all triggers.
func (g *Group) DeleteAll(ctx context.Context) error {
	var mu sync.Mutex
	var allErrors []error

	eg, gctx := errgroup.WithContext(ctx)
	eg.SetLimit(maxConcurrent)

	g.mu.Lock()
	defer g.mu.Unlock()

	for _, h := range g.configured {
		h := h

		eg.Go(func() error {
			select {
			case <-gctx.Done():
				return gctx.Err()
			default:
			}

			if err := g.Client.TriggerDel(gctx, h.Path, h.Name); err != nil {
				mu.Lock()
				allErrors = append(allErrors, err)
				mu.Unlock()
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		allErrors = append(allErrors, err)
	}

	return multierr.Combine(allErrors...)
}
