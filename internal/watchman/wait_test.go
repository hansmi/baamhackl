package watchman

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

type zeroTimer struct {
	timer *time.Timer
}

func (t *zeroTimer) Start(duration time.Duration) {
	t.timer = time.NewTimer(0)
}

func (t *zeroTimer) Stop() {
	if t.timer != nil {
		t.timer.Stop()
	}
}

func (t *zeroTimer) C() <-chan time.Time {
	return t.timer.C
}

var errPingFinal = errors.New("final error")
var errPingTest = errors.New("test error")

type fakePingClient []error

func (c *fakePingClient) Ping(ctx context.Context) error {
	if len(*c) > 0 {
		err := (*c)[0]
		*c = (*c)[1:]
		return err
	}

	return errPingFinal
}

func TestWaitForReady(t *testing.T) {
	for _, tc := range []struct {
		name    string
		ctx     context.Context
		c       pingClient
		b       backoff.BackOff
		wantErr error
	}{
		{
			name: "success",
			c:    &fakePingClient{nil},
			b:    backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Millisecond), 3),
		},
		{
			name:    "failure",
			c:       &fakePingClient{},
			b:       backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Millisecond), 3),
			wantErr: errPingFinal,
		},
		{
			name: "success after errors",
			c: &fakePingClient{
				errors.New("aaa"),
				errors.New("bbb"),
				errors.New("ccc"),
				errors.New("ddd"),
				nil,
			},
			b: backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Millisecond), 10),
		},
		{
			name: "retries exhausted",
			c: &fakePingClient{
				errors.New("aaa"),
				errors.New("bbb"),
				errors.New("ccc"),
				errPingTest,
				errors.New("ddd"),
			},
			b:       backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Millisecond), 3),
			wantErr: errPingTest,
		},
		{
			name: "canceled context",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			c:       &fakePingClient{},
			b:       backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Millisecond), 3),
			wantErr: context.Canceled,
		},
		{
			name: "context deadline",
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 0)
				cancel()
				return ctx
			}(),
			c:       &fakePingClient{},
			b:       backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Millisecond), 3),
			wantErr: context.DeadlineExceeded,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer zap.ReplaceGlobals(zaptest.NewLogger(t))()

			if tc.ctx == nil {
				tc.ctx = context.Background()
			}

			ctx, cancel := context.WithTimeout(tc.ctx, 10*time.Second)
			defer cancel()

			err := waitForReady(ctx, tc.c, tc.b, &zeroTimer{})

			if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Error diff (-want +got):\n%s", diff)
			}
		})
	}
}
