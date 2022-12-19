package watch

import (
	"context"
	"errors"
	"path/filepath"
	"sync"

	"github.com/hansmi/baamhackl/internal/config"
	"github.com/hansmi/baamhackl/internal/handlertask"
	"github.com/hansmi/baamhackl/internal/journal"
	"github.com/hansmi/baamhackl/internal/scheduler"
	"github.com/hansmi/baamhackl/internal/service"
	"github.com/hansmi/baamhackl/internal/waryio"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type handler struct {
	mu      sync.Mutex
	name    string
	cfg     *config.Handler
	pending map[string]*handlertask.Task
	journal *journal.Journal
	mc      *handlerMetricsCollector

	invoke func(context.Context, *handlertask.Task, func()) error
}

func newHandler(cfg *config.Handler) *handler {
	h := &handler{
		name:    cfg.Name,
		cfg:     cfg,
		journal: journal.New(cfg),
		pending: map[string]*handlertask.Task{},
		invoke: func(ctx context.Context, t *handlertask.Task, acquireLock func()) error {
			return t.Run(ctx, acquireLock)
		},
	}

	h.mc = newHandlerMetricsCollector(h)

	return h
}

func (h *handler) metrics() prometheus.Collector {
	return h.mc
}

func (h *handler) newTask(name string) *handlertask.Task {
	return handlertask.New(handlertask.Options{
		Config:  h.cfg,
		Journal: h.journal,
		Name:    name,
		Metrics: h.mc,
	})
}

func (h *handler) invokeTask(ctx context.Context, t *handlertask.Task) error {
	locked := false

	defer func() {
		if locked {
			h.mu.Unlock()
		}
	}()

	// Delay acquiring the handler lock until it's necessary.
	acquireLock := func() {
		if !locked {
			locked = true
			h.mu.Lock()
		}
	}

	err := h.invoke(ctx, t, acquireLock)

	if scheduler.AsTaskError(err).Permanent() {
		h.mc.ReportFinalTaskStatus(err)

		acquireLock()

		// Remove from pending tasks
		delete(h.pending, t.Name())
	} else {
		h.mc.ReportTaskRetry()
	}

	return err
}

func (h *handler) handle(sched *scheduler.Scheduler, req service.FileChangedRequest) error {
	logger := zap.L()

	if ok, err := waryio.SameStat(req.RootDir, h.cfg.Path); err != nil {
		return err
	} else if !ok {
		return errors.New("root directory in request differs from configuration")
	}

	name := filepath.Clean(req.Change.Name)

	h.mu.Lock()
	defer h.mu.Unlock()

	if t := h.pending[name]; t == nil {
		t = h.newTask(name)
		h.pending[name] = t
		sched.Add(func(ctx context.Context) error {
			return h.invokeTask(ctx, t)
		})
	} else {
		logger.Debug("File already in queue", zap.String("name", name))
	}

	h.mc.ReportFileChange()

	return nil
}

func (h *handler) prune(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.journal.Prune(ctx, zap.L())
}
