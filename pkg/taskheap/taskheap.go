package taskheap

import (
	"container/heap"

	"github.com/LambdaTest/neuron/pkg/core"
)

// Heap is a heap of tasks
type Heap []*core.TaskHeapItem

// Len returns the length of the heap
func (h Heap) Len() int {
	return len(h)
}

func (h Heap) Less(i, j int) bool {
	return h[i].TotalTestsDuration < h[j].TotalTestsDuration
}

// Swap swaps the values of two tasks
func (h Heap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

// Push adds a new task to the heap
func (h *Heap) Push(x interface{}) {
	item := x.(*core.TaskHeapItem)
	*h = append(*h, item)
}

// Pop removes the top task from the heap
func (h *Heap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	old[n-1] = nil // avoid memory leak
	*h = old[0 : n-1]
	return x
}

// UpdateHead updates the head task in the heap.
func (h *Heap) UpdateHead(testDuration int, testLocator string) {
	(*h)[0].TotalTestsDuration += testDuration
	(*h)[0].TestsAllocated += testLocator
	// heapify after updating the task
	heap.Fix(h, 0)
}
