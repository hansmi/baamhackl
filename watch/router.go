package watch

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/hansmi/baamhackl/internal/config"
	"github.com/hansmi/baamhackl/internal/fuzzduration"
	"github.com/hansmi/baamhackl/internal/scheduler"
	"github.com/hansmi/baamhackl/internal/service"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type router struct {
	handlerByName map[string]*handler
	sched         *scheduler.Scheduler
	pruneInterval time.Duration
}

func newRouter(handlers []*config.Handler) *router {
	r := &router{
		handlerByName: map[string]*handler{},
		sched:         scheduler.New(),
		pruneInterval: time.Hour,
	}

	for _, h := range handlers {
		r.handlerByName[h.Name] = newHandler(h)
	}

	return r
}

func (r *router) start(slots int) {
	r.sched.SetSlots(slots)
	r.sched.Start()
}

func (r *router) stop(ctx context.Context) error {
	return r.sched.Stop(ctx)
}

func (r *router) FileChanged(req service.FileChangedRequest) error {
	logger := zap.L()
	logger.Debug("Received file change", zap.Reflect("req", req))

	if req.HandlerName == "" || req.Change.Name == "" {
		return errors.New("missing handler name and/or changed file")
	}

	if !filepath.IsAbs(req.RootDir) {
		return fmt.Errorf("root directory must be an absolute path: %s", req.RootDir)
	}

	if filepath.IsAbs(req.Change.Name) {
		return fmt.Errorf("filename must be a relative path: %s", req.Change.Name)
	}

	h, ok := r.handlerByName[req.HandlerName]
	if !ok {
		return fmt.Errorf("handler %q not found", req.HandlerName)
	}

	return h.handle(r.sched, req)
}

func (r *router) startPruning(interval time.Duration) {
	r.pruneInterval = interval
	r.schedulePruning(interval / 10)
}

func (r *router) schedulePruning(after time.Duration) {
	r.sched.Add(r.pruneAll, scheduler.NextAfterDuration(fuzzduration.Random(after, 0.1)))
}

func (r *router) pruneAll(ctx context.Context) error {
	var allErrors error

	for _, h := range r.handlerByName {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		multierr.AppendInto(&allErrors, h.prune(ctx))
	}

	r.schedulePruning(r.pruneInterval)

	return allErrors
}
