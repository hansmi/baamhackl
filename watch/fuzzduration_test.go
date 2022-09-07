package watch

import (
	"testing"
	"time"
)

func TestFuzzDuration(t *testing.T) {
	for _, tc := range []struct {
		name   string
		d      time.Duration
		factor float32
		margin time.Duration
	}{
		{name: "zero"},
		{name: "hour", d: time.Hour},
		{name: "day", d: 24 * time.Hour},
		{
			name:   "hour, 10%",
			d:      time.Hour,
			factor: 0.1,
			margin: 3 * time.Minute,
		},
		{
			name:   "day, 1%",
			d:      24 * time.Hour,
			factor: 0.01,
			margin: 8 * time.Minute,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for i := 0; i < 100; i++ {
				got := fuzzDuration(tc.d, tc.factor)

				diff := got - tc.d
				if diff < 0 {
					diff = -diff
				}

				if diff > tc.margin {
					t.Errorf("fuzzDuration(%v, %v) returned %v (difference %v), want at most %v",
						tc.d, tc.factor, got, diff, tc.margin)
				}
			}
		})
	}
}
