package watch

import (
	"math/rand"
	"time"
)

// fuzzDuration returns the duration modified by a random amount in the range
// ±factor/2.
func fuzzDuration(d time.Duration, factor float32) time.Duration {
	fuzz := -(factor / 2) + (factor * rand.Float32())
	return d + time.Duration(fuzz*float32(d))
}
