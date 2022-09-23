package taskheap

import (
	"container/heap"
	"testing"

	"github.com/LambdaTest/neuron/pkg/core"
)

func TestHeapInit(t *testing.T) {
	var taskHeap Heap
	heap.Init(&taskHeap)
	if taskHeap.Len() != 0 {
		t.Errorf("Expected heap length to be 0, got %d", taskHeap.Len())
	}
}

func TestHeapPush(t *testing.T) {
	var taskHeap = Heap{&core.TaskHeapItem{
		TaskID:             "Task1",
		TotalTestsDuration: 3,
		TestsAllocated:     "test1",
	},
		&core.TaskHeapItem{
			TaskID:             "Task2",
			TotalTestsDuration: 2,
			TestsAllocated:     "test2",
		}}
	heap.Init(&taskHeap)
	heap.Push(&taskHeap, &core.TaskHeapItem{
		TaskID:             "Task3",
		TotalTestsDuration: 3,
		TestsAllocated:     "test3",
	})
	if taskHeap.Len() != 3 {
		t.Errorf("Expected heap length to be 3, got %d", taskHeap.Len())
	}
}

func TestHeapPop(t *testing.T) {
	var taskHeap = Heap{&core.TaskHeapItem{
		TaskID:             "Task1",
		TotalTestsDuration: 3,
		TestsAllocated:     "test1",
	},
		&core.TaskHeapItem{
			TaskID:             "Task2",
			TotalTestsDuration: 2,
			TestsAllocated:     "test2",
		},
		&core.TaskHeapItem{
			TaskID:             "Task3",
			TotalTestsDuration: 1,
			TestsAllocated:     "test3",
		},
	}
	heap.Init(&taskHeap)
	x := heap.Pop(&taskHeap)

	if taskHeap.Len() != 2 {
		t.Errorf("Expected heap length to be 2, got %d", taskHeap.Len())
	}

	head := x.(*core.TaskHeapItem)

	if head.TaskID != "Task3" {
		t.Errorf("Expected TaskID to be Task3, got %s", head.TaskID)
	}
}

func TestHeapUpdateHead(t *testing.T) {
	var taskHeap = Heap{&core.TaskHeapItem{
		TaskID:             "Task1",
		TotalTestsDuration: 3,
		TestsAllocated:     "test1",
	},
		&core.TaskHeapItem{
			TaskID:             "Task2",
			TotalTestsDuration: 2,
			TestsAllocated:     "test2",
		},
		&core.TaskHeapItem{
			TaskID:             "Task3",
			TotalTestsDuration: 1,
			TestsAllocated:     "test3",
		},
	}
	heap.Init(&taskHeap)
	taskHeap.UpdateHead(7, "test4")
	x := heap.Pop(&taskHeap)
	if taskHeap.Len() != 2 {
		t.Errorf("Expected heap length to be 2, got %d", taskHeap.Len())
	}
	head := x.(*core.TaskHeapItem)

	if head.TaskID != "Task2" {
		t.Errorf("Expected TaskID to be Task2, got %s", head.TaskID)
	}
}
