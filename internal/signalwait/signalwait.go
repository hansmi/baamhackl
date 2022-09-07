package signalwait

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"

	"golang.org/x/sys/unix"
)

// ErrSignal is wrapped by the errors returned by WaitFunc when a signal has
// been received.
var ErrSignal = errors.New("received signal")

// WaitFunc is the type of functions waiting for signals.
type WaitFunc func(context.Context) error

type waiter struct {
	ch     chan os.Signal
	active chan struct{}
}

func setup(signals []os.Signal) *waiter {
	w := &waiter{
		ch:     make(chan os.Signal, 1),
		active: make(chan struct{}),
	}

	runtime.SetFinalizer(w, (*waiter).stop)

	signal.Notify(w.ch, signals...)

	return w
}

// Setup starts relaying incoming signals into an internal buffer. The returned
// wait function sleeps until either a signal has been received (which may have
// happened before the wait function is called) or the wait function's context
// is canceled. The stop function causes signals to not be captured anymore.
func Setup(signals ...os.Signal) (wait WaitFunc, stop context.CancelFunc) {
	w := setup(signals)
	return w.wait, w.stop
}

func (w *waiter) stop() {
	signal.Stop(w.ch)

	select {
	case <-w.active:
	default:
		close(w.active)
	}
}

func (w *waiter) wait(ctx context.Context) error {
	select {
	case <-w.active:
		return nil

	case sig := <-w.ch:
		var suffix string

		if num, ok := sig.(unix.Signal); ok && num != 0 {
			suffix = fmt.Sprintf(" (number %d)", num)
		}

		return fmt.Errorf("%w: %q%s", ErrSignal, sig, suffix)

	case <-ctx.Done():
		return ctx.Err()
	}
}
