package watch

import (
	"context"
	"time"

	"go.uber.org/multierr"
)

type cleanupGroup []func(context.Context) error

func (c *cleanupGroup) append(fn func(context.Context) error) {
	*c = append(*c, fn)
}

func (c cleanupGroup) CallWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return c.Call(ctx)
}

func (c cleanupGroup) Call(ctx context.Context) error {
	var allErrors error

	for i := len(c) - 1; i >= 0; i-- {
		multierr.AppendInto(&allErrors, c[i](ctx))
	}

	return allErrors
}
