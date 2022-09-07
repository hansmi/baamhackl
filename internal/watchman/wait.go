package watchman

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
	"go.uber.org/zap"
)

type pingClient interface {
	Ping(context.Context) error
}

func waitForReady(ctx context.Context, c pingClient, b backoff.BackOff, timer backoff.Timer) error {
	errCount := 0

	return backoff.RetryNotifyWithTimer(func() error {
		if err := c.Ping(ctx); err != nil {
			if errCount == 3 {
				zap.L().Info("Waiting for Watchman to become available", zap.Error(err))
			}

			errCount++

			return err
		}

		return nil
	}, backoff.WithContext(b, ctx), nil, timer)
}

func WaitForReady(ctx context.Context, c Client) error {
	b := backoff.NewExponentialBackOff()
	b.RandomizationFactor = 0.1
	b.MaxInterval = time.Second
	b.MaxElapsedTime = 0

	return waitForReady(ctx, c, b, nil)
}
