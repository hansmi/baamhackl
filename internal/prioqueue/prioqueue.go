package prioqueue

import (
	"container/heap"
	"fmt"
	"math"
)

const undefinedIndex int = math.MinInt

// LessFunc returns whether the first parameter is to be considered less than
// the second parameter.
type LessFunc func(any, any) bool

// Item contains a reference to an enqueued value.
type Item struct {
	Value any
	index int
}

type heapAdapter struct {
	*PrioQueue
}

var _ heap.Interface = (*heapAdapter)(nil)

func (s heapAdapter) Len() int {
	return len(s.items)
}

func (s heapAdapter) Less(i, j int) bool {
	return s.PrioQueue.Less(s.items[i].Value, s.items[j].Value)
}

func (s heapAdapter) Swap(i, j int) {
	s.items[i], s.items[j] = s.items[j], s.items[i]
	s.items[i].index = i
	s.items[j].index = j
}

func (s heapAdapter) Push(v any) {
	item := v.(*Item)

	if item.index != undefinedIndex {
		panic(fmt.Sprintf("item already part of a heap: %#v", item))
	}

	s.items = append(s.items, item)

	item.index = len(s.items) - 1
}

func (s heapAdapter) Pop() any {
	n := len(s.items)
	item := s.items[n-1]

	// Avoid memory leak
	s.items[n-1] = nil

	s.items = s.items[0 : n-1]

	item.index = undefinedIndex

	return item
}

// PrioQueue is a min-heap queue with items sorted using the Less function.
type PrioQueue struct {
	Less  LessFunc
	items []*Item
}

// Len returns the number of items in the queue.
func (q *PrioQueue) Len() int {
	return len(q.items)
}

// Push inserts a new value into the queue, returning an Item pointer suitable
// for updating or removing the value directly.
func (q *PrioQueue) Push(value any) *Item {
	item := &Item{
		Value: value,
		index: undefinedIndex,
	}

	heap.Push(heapAdapter{q}, item)

	return item
}

// Pop removes and returns the minimum value from the queue.
func (q *PrioQueue) Pop() any {
	if len(q.items) == 0 {
		return nil
	}

	return heap.Pop(heapAdapter{q}).(*Item).Value
}

// Peek returns the minimum value in the queue.
func (q *PrioQueue) Peek() any {
	if len(q.items) == 0 {
		return nil
	}

	return q.items[0].Value
}

// Clear removes all items from the queue.
func (q *PrioQueue) Clear() {
	for count := len(q.items); count > 0; count-- {
		q.Pop()
	}
}

// Update corrects item's position in the queue after its underlying value has
// been changed.
func (q *PrioQueue) Update(item *Item) {
	heap.Fix(heapAdapter{q}, item.index)
}

// Remove removes an item regardless of its current position in the queue.
func (q *PrioQueue) Remove(item *Item) {
	heap.Remove(heapAdapter{q}, item.index)
}
