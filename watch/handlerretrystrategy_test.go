package watch

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hansmi/baamhackl/internal/config"
	"github.com/hansmi/baamhackl/internal/scheduler"
)

func TestHandlerRetryStrategy(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  config.Handler
		want []time.Duration
	}{
		{name: "empty"},
		{
			name: "defaults",
			cfg:  config.HandlerDefaults,
			want: []time.Duration{
				15 * time.Minute,
				22*time.Minute + 30*time.Second,
			},
		},
		{
			name: "max",
			cfg: config.Handler{
				RetryCount:        5,
				RetryDelayInitial: time.Minute,
				RetryDelayFactor:  2,
				RetryDelayMax:     3 * time.Minute,
			},
			want: []time.Duration{
				time.Minute,
				2 * time.Minute,
				3 * time.Minute,
				3 * time.Minute,
				3 * time.Minute,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := append([]time.Duration(nil), tc.want...)

			// Always test a few more rounds
			for i := 0; i < 10; i++ {
				want = append(want, scheduler.Stop)
			}

			r := newHandlerRetryStrategy(tc.cfg)

			var got []time.Duration

			for range want {
				got = append(got, r.current())
				r.advance()
			}

			if diff := cmp.Diff(want, got, cmpopts.EquateApproxTime(time.Millisecond)); diff != "" {
				t.Errorf("Delay diff (-want +got):\n%s", diff)
			}
		})
	}
}
