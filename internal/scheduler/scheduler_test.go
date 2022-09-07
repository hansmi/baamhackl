package scheduler

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hansmi/baamhackl/internal/testutil"
	"github.com/jonboulle/clockwork"
)

func TestEmpty(t *testing.T) {
	for _, quiesce := range []bool{false, true} {
		t.Run(fmt.Sprint(quiesce), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			t.Cleanup(cancel)

			s := New()
			s.Start()

			if quiesce {
				if err := s.Quiesce(ctx); err != nil {
					t.Errorf("Quiesce() failed: %v", err)
				}
			}

			if err := s.Stop(ctx); err != nil {
				t.Errorf("Stop() failed: %v", err)
			}
		})
	}
}

func TestStartMultipleTimes(t *testing.T) {
	for _, afterStop := range []bool{false, true} {
		t.Run(fmt.Sprint(afterStop), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			t.Cleanup(cancel)

			s := New()
			s.Start()

			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Start() did not panic")
				} else if diff := cmp.Diff(r, "Scheduler may only start once"); diff != "" {
					t.Errorf("Panic diff (-want +got):\n%s", diff)
				}
			}()

			if !afterStop {
				s.Start()
			}

			if err := s.Stop(ctx); err != nil {
				t.Errorf("Stop() failed: %v", err)
			}

			if afterStop {
				s.Start()
			}
		})
	}
}

func TestAddFromRunningTask(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	var numbers [200]int

	for idx := range numbers {
		numbers[idx] = idx
	}

	rand.Shuffle(len(numbers), func(a, b int) {
		numbers[a], numbers[b] = numbers[b], numbers[a]
	})

	s := New()
	s.SetSlots(3)
	s.Start()

	addNoop := func() {
		s.Add(func(context.Context) error {
			return nil
		})
	}

	tasksAdded := make(chan struct{})
	taskDone := make([]bool, len(numbers))

	s.Add(func(context.Context) error {
		defer close(tasksAdded)

		for i := 0; i < 10; i++ {
			addNoop()
		}

		for _, i := range numbers {
			i := i

			addNoop()
			s.Add(func(context.Context) error {
				taskDone[i] = true
				return nil
			})
			addNoop()
		}

		return nil
	})

	<-tasksAdded

	if err := s.Quiesce(ctx); err != nil {
		t.Errorf("Quiesce() failed: %v", err)
	}

	if err := s.Stop(ctx); err != nil {
		t.Errorf("Stop() failed: %v", err)
	}

	for idx := range taskDone {
		if !taskDone[idx] {
			t.Errorf("Task %v didn't run", idx)
		}
	}
}

func TestRunOrder(t *testing.T) {
	const count = 127

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	s := New()

	// Must not use multiple slots as one may start a later task faster than
	// another slot running an earlier task.
	s.SetSlots(1)

	var mu sync.Mutex
	var got []int

	for i := 0; i < count; i++ {
		i := i

		s.Add(func(context.Context) error {
			mu.Lock()
			defer mu.Unlock()

			got = append(got, i)

			return nil
		})
	}

	s.Start()

	if err := s.Quiesce(ctx); err != nil {
		t.Errorf("Quiesce() failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if err := s.Stop(ctx); err != nil {
		t.Errorf("Stop() failed: %v", err)
	}

	want := []int{}

	for i := 0; i < count; i++ {
		want = append(want, i)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Run order mismatch (-want +got):\n%s", diff)
	}
}

func TestRunMany(t *testing.T) {
	for _, slots := range []int{3, 10} {
		for _, taskCount := range []int32{11, 123, 500} {
			t.Run(fmt.Sprintf("slots=%d,count=%d", slots, taskCount), func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				t.Cleanup(cancel)

				var count int32

				s := New()
				s.SetSlots(slots)
				s.Start()

				for i := int32(0); i < taskCount; i++ {
					s.Add(func(context.Context) error {
						time.Sleep(time.Millisecond)
						atomic.AddInt32(&count, 1)
						return nil
					})
				}

				if err := s.Quiesce(ctx); err != nil {
					t.Errorf("Quiesce() failed: %v", err)
				}

				if err := s.Stop(ctx); err != nil {
					t.Errorf("Stop() failed: %v", err)
				}

				if diff := cmp.Diff(taskCount, atomic.LoadInt32(&count)); diff != "" {
					t.Errorf("Task count diff (-want +got):\n%s", diff)
				}
			})
		}
	}
}

func TestFailure(t *testing.T) {
	// t.Cleanup(zap.ReplaceGlobals(zaptest.NewLogger(t)))

	var counter int32

	var previousRun time.Time
	var previousDelay time.Duration

	s := New()
	s.Add(func(context.Context) error {
		now := time.Now()

		if !previousRun.IsZero() {
			if earliest := previousRun.Add(previousDelay); now.Before(earliest) {
				t.Errorf("Task invoked too soon (%s < %s)", now, earliest)
			}
		}

		count := atomic.AddInt32(&counter, 1)

		err := fmt.Errorf("test error %d", count)

		if count < 100 {
			err = &TaskError{
				Err:        err,
				RetryDelay: time.Duration(count/10) * time.Millisecond,
			}
			previousDelay = err.(*TaskError).RetryDelay
		} else {
			previousDelay = 0
		}

		previousRun = now

		return err
	})
	s.Start()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	if err := s.Quiesce(ctx); err != nil {
		t.Errorf("Quiesce() failed: %v", err)
	}

	if err := s.Stop(ctx); err != nil {
		t.Errorf("Stop() failed: %v", err)
	}
}

func TestNextAfterDuration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	fc := clockwork.NewFakeClock()
	testutil.ReplaceClock(t, &clock, fc)

	done := make(chan struct{})

	s := New()
	s.Add(func(context.Context) error {
		close(done)
		return nil
	}, NextAfterDuration(time.Minute))
	s.Start()

	fc.BlockUntil(1)
	fc.Advance(time.Minute + time.Second)

	select {
	case <-done:
	case <-ctx.Done():
		t.Errorf("Task didn't run on time: %v", ctx.Err())
	}

	if err := s.Stop(ctx); err != nil {
		t.Errorf("Stop() failed: %v", err)
	}
}
