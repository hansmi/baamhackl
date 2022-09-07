package uniquename

import (
	"math/rand"
	"time"

	"github.com/jonboulle/clockwork"
)

type Runtime struct {
	Clock interface {
		Now() time.Time
	}
	Loc     *time.Location
	RandInt func() int32
}

var DefaultRuntime = Runtime{
	Clock:   clockwork.NewRealClock(),
	Loc:     time.Local,
	RandInt: rand.Int31,
}

var globalRuntime = DefaultRuntime

func SetRuntime(r Runtime) func() {
	original := globalRuntime
	globalRuntime = r
	return func() {
		globalRuntime = original
	}
}
