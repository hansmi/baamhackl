package testutil

import (
	"testing"

	"github.com/jonboulle/clockwork"
)

func ReplaceClock(t *testing.T, clock *clockwork.Clock, override clockwork.Clock) {
	t.Helper()

	orig := *clock
	*clock = override

	t.Cleanup(func() {
		*clock = orig
	})
}
