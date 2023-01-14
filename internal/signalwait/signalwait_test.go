package signalwait

import (
	"context"
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hansmi/baamhackl/internal/testutil"
)

var sigs = []os.Signal{
	syscall.SIGUSR1,
	syscall.SIGUSR2,
	syscall.SIGWINCH,
}

// sendToSelf sends a signal to the current process.
func sendToSelf(t *testing.T, sig os.Signal) {
	t.Helper()

	if err := testutil.Raise(sig); err != nil {
		t.Fatal(err)
	}
}

func checkWait(ctx context.Context, t *testing.T, fn WaitFunc, wantErr error) {
	t.Helper()

	ctx, cancelCtx := context.WithTimeout(ctx, 10*time.Second)
	t.Cleanup(cancelCtx)

	err := fn(ctx)

	if diff := cmp.Diff(wantErr, err, cmpopts.EquateErrors()); diff != "" {
		t.Errorf("Wait error diff (-want +got):\n%s", diff)
	}
}

func setupForTest(t *testing.T, sig os.Signal) (func(context.Context, error), context.CancelFunc) {
	t.Helper()

	wait, stop := Setup(sig)
	t.Cleanup(stop)

	return func(ctx context.Context, wantErr error) {
		checkWait(ctx, t, wait, wantErr)
	}, stop
}

func TestSignalReceived(t *testing.T) {
	for _, sig := range sigs {
		wait, _ := setupForTest(t, sig)
		sendToSelf(t, sig)
		wait(context.Background(), ErrSignal)
	}
}

func TestStopEarly(t *testing.T) {
	for _, sig := range sigs {
		wait, stop := setupForTest(t, sig)
		stop()
		wait(context.Background(), nil)
	}
}

func TestStopManyTimes(t *testing.T) {
	for _, sig := range sigs {
		wait, stop := setupForTest(t, sig)
		for i := 0; i < 10; i++ {
			stop()
		}
		wait(context.Background(), nil)
	}
}

func TestCanceledContext(t *testing.T) {
	for _, sig := range sigs {
		wait, _ := setupForTest(t, sig)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		wait(ctx, context.Canceled)
	}
}

func TestDeadlineExceeded(t *testing.T) {
	for _, sig := range sigs {
		wait, _ := setupForTest(t, sig)
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()
		wait(ctx, context.DeadlineExceeded)
	}
}

func TestFinalizer(t *testing.T) {
	w := setup(sigs)

	active := w.active

	sendToSelf(t, sigs[0])
	checkWait(context.Background(), t, w.wait, ErrSignal)

	select {
	case <-active:
		t.Error("Waiter is not active")
	default:
	}

	w = nil

	// Trigger finalizer
	runtime.GC()

	select {
	case value, ok := <-active:
		if ok {
			t.Errorf("Received unexpected value: %v", value)
		}

	case <-time.After(5 * time.Second):
		t.Errorf("Finalizer didn't run")
	}
}
