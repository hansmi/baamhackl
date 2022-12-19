package watch

import (
	"math"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// makeSecondsBuckets creates exponentially increasing histogram buckets for
// full seconds. If the time between min and max is not enough there will be
// fewer than count buckets.
func makeSecondsBuckets(min, max time.Duration, count int) []float64 {
	buckets := prometheus.ExponentialBucketsRange(min.Seconds(), max.Seconds(), count)

	dst := 0

	for _, i := range buckets {
		if v := math.Round(i); dst == 0 || v > buckets[dst-1] {
			buckets[dst] = v
			dst++
		}
	}

	return buckets[:dst]
}
