package cleanupgroup

import (
	"context"
	"time"

	"go.uber.org/multierr"
)

type CleanupGroup []func(context.Context) error

func (c *CleanupGroup) Append(fn func(context.Context) error) {
	*c = append(*c, fn)
}

func (c CleanupGroup) CallWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return c.Call(ctx)
}

func (c CleanupGroup) Call(ctx context.Context) error {
	var allErrors error

	for i := len(c) - 1; i >= 0; i-- {
		multierr.AppendInto(&allErrors, c[i](ctx))
	}

	return allErrors
}
