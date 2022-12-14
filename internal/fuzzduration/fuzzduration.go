package fuzzduration

import (
	"math/rand"
	"time"
)

// Random returns the duration modified by a random amount in the range
// ±factor/2.
func Random(d time.Duration, factor float32) time.Duration {
	fuzz := -(factor / 2) + (factor * rand.Float32())
	return d + time.Duration(fuzz*float32(d))
}
