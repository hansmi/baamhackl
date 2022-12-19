package watch

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestMakeSecondsBuckets(t *testing.T) {
	for _, tc := range []struct {
		name  string
		min   time.Duration
		max   time.Duration
		count int
		want  []float64
	}{
		{
			name:  "one",
			min:   time.Second,
			max:   time.Minute,
			count: 1,
			want:  []float64{1},
		},
		{
			name:  "two",
			min:   time.Second,
			max:   time.Minute,
			count: 2,
			want:  []float64{1, 60},
		},
		{
			name:  "hour",
			min:   time.Second,
			max:   time.Hour,
			count: 10,
			want:  []float64{1, 2, 6, 15, 38, 95, 235, 583, 1449, 3600},
		},
		{
			name:  "nanosecond",
			min:   time.Millisecond,
			max:   10 * time.Minute,
			count: 10,
			want:  []float64{0, 2, 7, 31, 137, 600},
		},
		{
			name:  "overlapping values",
			min:   time.Second,
			max:   10 * time.Second,
			count: 30,
			want:  []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := makeSecondsBuckets(tc.min, tc.max, tc.count)

			if diff := cmp.Diff(tc.want, got, cmpopts.EquateApprox(0, 0.1)); diff != "" {
				t.Errorf("Bucket diff (-want +got):\n%s", diff)
			}
		})
	}
}
