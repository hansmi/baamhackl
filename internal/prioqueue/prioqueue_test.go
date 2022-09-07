package prioqueue

import "testing"

type process struct {
	priority int
}

func newProcessQueue() *PrioQueue {
	return &PrioQueue{
		Less: func(a any, b any) bool {
			return a.(*process).priority < b.(*process).priority
		},
	}
}

func TestEmpty(t *testing.T) {
	q := newProcessQueue()

	if got := q.Len(); got != 0 {
		t.Errorf("Len() returned %d, want 0", got)
	}

	if got := q.Pop(); got != nil {
		t.Errorf("Pop() returned %v, want nil", got)
	}

	if got := q.Peek(); got != nil {
		t.Errorf("Peek() returned %v, want nil", got)
	}
}

func TestPushPop(t *testing.T) {
	q := newProcessQueue()

	q.Push(&process{priority: 102})
	q.Push(&process{priority: 100})
	q.Push(&process{priority: 99})
	q.Push(&process{priority: 101})

	if got, want := q.Len(), 4; got != want {
		t.Errorf("Len() returned %v, want %d", got, want)
	}

	for _, want := range []int{99, 100, 101, 102} {
		if got := q.Pop().(*process).priority; got != want {
			t.Errorf("Pop() returned priority %d, want %d", got, want)
		}
	}

	if got := q.Len(); got != 0 {
		t.Errorf("Len() returned %d, want 0", got)
	}
}

func TestPushUpdatePop(t *testing.T) {
	q := newProcessQueue()

	item := q.Push(&process{priority: 100})

	q.Push(&process{priority: 200})

	item.Value.(*process).priority = 300

	q.Update(item)

	for _, want := range []int{200, 300} {
		if got := q.Pop().(*process).priority; got != want {
			t.Errorf("Pop() returned priority %d, want %d", got, want)
		}
	}

	if got := q.Len(); got != 0 {
		t.Errorf("Len() returned %d, want 0", got)
	}
}

func TestRemove(t *testing.T) {
	q := newProcessQueue()

	q.Push(&process{priority: -1})
	second := q.Push(&process{priority: -2})
	q.Push(&process{priority: -3})

	if got, want := q.Len(), 3; got != want {
		t.Errorf("Len() returned %v, want %d", got, want)
	}

	q.Remove(second)

	for _, want := range []int{-3, -1} {
		if got := q.Pop().(*process).priority; got != want {
			t.Errorf("Pop() returned priority %d, want %d", got, want)
		}
	}

	if got := q.Len(); got != 0 {
		t.Errorf("Len() returned %d, want 0", got)
	}
}
