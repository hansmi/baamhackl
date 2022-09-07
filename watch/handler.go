package watch

import (
	"context"
	"errors"
	"path/filepath"
	"sync"

	"github.com/hansmi/baamhackl/internal/config"
	"github.com/hansmi/baamhackl/internal/scheduler"
	"github.com/hansmi/baamhackl/internal/service"
	"github.com/hansmi/baamhackl/internal/waryio"
	"go.uber.org/zap"
)

type handler struct {
	mu      sync.Mutex
	name    string
	cfg     *config.Handler
	pending map[string]*handlerTask
	journal *journal

	invoke func(context.Context, *handlerTask, func()) error
}

func newHandler(cfg *config.Handler) *handler {
	return &handler{
		name:    cfg.Name,
		cfg:     cfg,
		journal: newJournal(cfg),
		pending: map[string]*handlerTask{},
		invoke: func(ctx context.Context, t *handlerTask, acquireLock func()) error {
			return t.run(ctx, acquireLock)
		},
	}
}

func (h *handler) newTask(name string) *handlerTask {
	return &handlerTask{
		cfg:        h.cfg,
		journal:    h.journal,
		name:       name,
		fuzzFactor: 0.1,
	}
}

func (h *handler) invokeTask(ctx context.Context, t *handlerTask) error {
	locked := false

	defer func() {
		if locked {
			h.mu.Unlock()
		}
	}()

	acquireLock := func() {
		if !locked {
			locked = true
			h.mu.Lock()
		}
	}

	err := h.invoke(ctx, t, acquireLock)

	if scheduler.AsTaskError(err).Permanent() {
		acquireLock()

		// Remove from pending tasks
		delete(h.pending, t.name)
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

	return nil
}

func (h *handler) prune(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.journal.prune(ctx, zap.L())
}
