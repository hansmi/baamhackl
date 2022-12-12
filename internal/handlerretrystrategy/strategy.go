package handlerretrystrategy

import (
	"math"
	"time"

	"github.com/hansmi/baamhackl/internal/config"
	"github.com/hansmi/baamhackl/internal/scheduler"
)

type Strategy struct {
	remaining int
	delay     time.Duration
	factor    float64
	max       time.Duration
}

func New(cfg config.Handler) *Strategy {
	return &Strategy{
		remaining: cfg.RetryCount,
		delay:     cfg.RetryDelayInitial,
		factor:    cfg.RetryDelayFactor,
		max:       cfg.RetryDelayMax,
	}
}

func (r *Strategy) Advance() {
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

func (r *Strategy) Current() time.Duration {
	if r.remaining <= 0 {
		return scheduler.Stop
	}

	return r.delay
}
