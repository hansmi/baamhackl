package watch

import (
	"math"
	"time"

	"github.com/hansmi/baamhackl/internal/config"
	"github.com/hansmi/baamhackl/internal/scheduler"
)

type handlerRetryStrategy struct {
	remaining int
	delay     time.Duration
	factor    float64
	max       time.Duration
}

func newHandlerRetryStrategy(cfg config.Handler) *handlerRetryStrategy {
	return &handlerRetryStrategy{
		remaining: cfg.RetryCount,
		delay:     cfg.RetryDelayInitial,
		factor:    cfg.RetryDelayFactor,
		max:       cfg.RetryDelayMax,
	}
}

func (r *handlerRetryStrategy) advance() {
	if r.remaining < 0 {
		return
	}

	d := float64(r.delay) * r.factor

	if r.max > 0 {
		d = math.Min(d, float64(r.max))
	}

	r.delay = time.Duration(d)
	r.remaining--
}

func (r *handlerRetryStrategy) current() time.Duration {
	if r.remaining <= 0 {
		return scheduler.Stop
	}

	return r.delay
}
