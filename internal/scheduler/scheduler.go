package scheduler

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hansmi/baamhackl/internal/prioqueue"
	"github.com/jonboulle/clockwork"
)

var seqCounter int64

func nextSequenceNumber() int64 {
	seq := atomic.AddInt64(&seqCounter, 1)

	if seq == 0 {
		return nextSequenceNumber()
	}

	return seq
}

type Scheduler struct {
	mu sync.Mutex

	slots int

	// Tasks sorted by insertion order.
	tasksByOrder prioqueue.PrioQueue

	// Tasks sorted by due time.
	tasksByTime prioqueue.PrioQueue

	loopActiveCount   int32
	loopStopRequested bool
	loopNotification  event
	loopFinished      event

	taskContext       context.Context
	taskContextCancel context.CancelFunc
	taskActiveCount   int
	taskFinished      event
}

// New creates a new scheduler instance. The slot count defaults to the number
// CPUs in the system.
func New() *Scheduler {
	s := &Scheduler{
		slots: runtime.NumCPU(),

		tasksByOrder: prioqueue.PrioQueue{
			Less: func(a, b interface{}) bool {
				lhs := a.(*Task)
				rhs := b.(*Task)

				return (lhs.seq - rhs.seq) < 0
			},
		},
		tasksByTime: prioqueue.PrioQueue{
			Less: func(a, b interface{}) bool {
				lhs := a.(*Task)
				rhs := b.(*Task)

				return lhs.nextAfter.Before(rhs.nextAfter)
			},
		},

		loopNotification: newEvent(),
		loopFinished:     newEvent(),

		taskFinished: newEvent(),
	}

	s.taskContext, s.taskContextCancel = context.WithCancel(context.Background())

	return s
}

// SetSlots changes the number of tasks running concurrently.
func (s *Scheduler) SetSlots(count int) {
	if count < 1 {
		count = 1
	}

	s.mu.Lock()
	s.slots = count
	s.mu.Unlock()

	s.loopNotification.Set()
}

// enqueue adds a task to the appropriate queue. If a due time is set the
// time-sorted queue is used, the insertion order queue otherwise.
func (s *Scheduler) enqueue(t *Task) {
	if t.seq == 0 {
		panic("task lacks a sequence number")
	}

	if !(t.seqItem == nil && t.timeItem == nil) {
		panic("task already in queue")
	}

	if t.nextAfter.IsZero() {
		t.seqItem = s.tasksByOrder.Push(t)
	} else {
		t.timeItem = s.tasksByTime.Push(t)
	}
}

// ScheduleOption is a function configuring a task.
type ScheduleOption func(*Task)

// NextAfterDuration configures the task to only run after the given amount of
// wall time.
func NextAfterDuration(d time.Duration) ScheduleOption {
	return func(t *Task) {
		t.nextAfter = clock.Now().Add(d)
	}
}

// Add a new task function to the scheduler. Unless configured otherwise
// through an option tasks are started in the order they're added.
func (s *Scheduler) Add(fn TaskFunc, opts ...ScheduleOption) {
	if fn == nil {
		panic("Function is nil")
	}

	t := &Task{
		fn:  fn,
		seq: nextSequenceNumber(),
	}

	// if !(t.seq == 0 && t.seqItem == nil && t.timeItem == nil) {
	// 	panic("tasks can't be reused")
	// }

	for _, opt := range opts {
		opt(t)
	}

	s.mu.Lock()
	s.enqueue(t)
	s.mu.Unlock()

	s.loopNotification.Set()
}

// Check whether there's a task waiting to be run. The task is removed from its
// queue. Returns nil if no task is ready to run. In that case the second
// return value is the amount of time to wait before the next task is due.
func (s *Scheduler) popNextLocked() (*Task, time.Duration) {
	remaining := time.Duration(-1)

	if nextByTime := s.tasksByTime.Peek(); nextByTime != nil {
		remaining = nextByTime.(*Task).nextAfter.Sub(clock.Now())

		if remaining <= 0 {
			runtask := s.tasksByTime.Pop().(*Task)
			runtask.timeItem = nil

			return runtask, -1
		}
	}

	var runtask *Task

	if nextByOrder := s.tasksByOrder.Pop(); nextByOrder != nil {
		runtask = nextByOrder.(*Task)
		runtask.seqItem = nil
	}

	return runtask, remaining
}

func (s *Scheduler) loop() {
	var nextTaskTimer clockwork.Timer

	for {
		s.mu.Lock()
		if s.loopStopRequested {
			s.mu.Unlock()
			return
		}

		var runtask *Task
		var nextTaskDueCh <-chan time.Time

		if s.taskActiveCount < s.slots {
			var remaining time.Duration

			runtask, remaining = s.popNextLocked()

			if remaining > 0 {
				if nextTaskTimer == nil {
					nextTaskTimer = clock.NewTimer(remaining)
				} else {
					if !nextTaskTimer.Stop() {
						select {
						case <-nextTaskTimer.Chan():
						default:
						}
					}

					nextTaskTimer.Reset(remaining)
				}

				nextTaskDueCh = nextTaskTimer.Chan()
			}
		}

		if runtask == nil {
			s.mu.Unlock()

			// Wait for next task
			select {
			case <-nextTaskDueCh:
			case <-s.loopNotification.Chan():
			}

			continue
		}

		// Launch new task
		go func() {
			s.taskActiveCount++
			s.mu.Unlock()

			s.runTask(runtask)
		}()
	}
}

func (s *Scheduler) runTask(t *Task) {
	finished := t.run(s.taskContext)

	s.mu.Lock()
	s.taskActiveCount--

	if !finished {
		s.enqueue(t)
	}

	if s.tasksByOrder.Len() > 0 || s.tasksByTime.Len() > 0 {
		s.loopNotification.Set()
	}
	s.mu.Unlock()

	s.taskFinished.Set()
}

// Start launches the scheduler loop in a separate goroutine before returning.
// Tasks already in the queue will be run in the same order as they would be if
// they were added later.
func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.loopStopRequested || s.loopActiveCount != 0 {
		s.mu.Unlock()
		panic("Scheduler may only start once")
	}

	go func() {
		s.loopActiveCount++
		s.mu.Unlock()

		defer func() {
			s.mu.Lock()
			s.loopActiveCount--
			s.mu.Unlock()

			s.loopFinished.Set()
		}()

		s.loop()
	}()
}

// Quiesce waits until all tasks have been run or the context is cancelled.
// Tasks are not affected on context cancellation. If tasks are added
// concurrently the behaviour is unspecified.
func (s *Scheduler) Quiesce(ctx context.Context) error {
	s.loopNotification.Set()

	for {
		s.mu.Lock()
		if s.taskActiveCount == 0 && s.tasksByOrder.Len() == 0 && s.tasksByTime.Len() == 0 {
			s.mu.Unlock()
			return nil
		}
		s.mu.Unlock()

		select {
		case <-s.loopFinished.Chan():
		case <-s.taskFinished.Chan():
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Stop waits until currently running tasks have finished or the context is
// cancelled. The latter will also cancel the context for running tasks. Tasks
// not yet started will be abandoned.
func (s *Scheduler) Stop(ctx context.Context) error {
	defer s.taskContextCancel()

	s.mu.Lock()
	s.loopStopRequested = true
	s.mu.Unlock()

	// Wake up loop so it can terminate
	s.loopNotification.Set()

	var err error

	ctxDoneChan := ctx.Done()

	for {
		s.mu.Lock()
		if s.loopActiveCount == 0 && s.taskActiveCount == 0 {
			s.tasksByOrder.Clear()
			s.tasksByTime.Clear()
			s.mu.Unlock()
			break
		}
		s.mu.Unlock()

		select {
		case <-s.loopFinished.Chan():
		case <-s.taskFinished.Chan():
		case <-ctxDoneChan:
			err = ctx.Err()
			ctxDoneChan = nil

			// Cancel active tasks
			s.taskContextCancel()
		}
	}

	return err
}
