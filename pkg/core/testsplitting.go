package core

import (
	"context"
)

// SplitMode is the mode for splitting tests
type SplitMode string

// list of support test splitting modes
const (
	FileSplit SplitMode = "file"
	TestSplit SplitMode = "test"
)

// TaskHeapItem is an item in the task heap.
type TaskHeapItem struct {
	TaskID             string
	TestsAllocated     string
	TotalTestsDuration int
}

type LocatorConfig struct {
	Locator string `json:"locator"`
}

type InputLocatorConfig struct {
	Locators []LocatorConfig `json:"locators"`
}

// TestSplitter splits the impacted tests into multiple tasks.
type TestSplitter interface {
	// Split will split the impacted tests into multiple tasks.
	Split(ctx context.Context, orgID, buildID, discoveryTaskID string,
		testIDs []string, executeAll bool, parallelCount int,
		branch string, splitMode SplitMode, subModule string) error
}
