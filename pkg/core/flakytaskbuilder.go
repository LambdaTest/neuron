package core

import "context"

// FlakyTaskBuilder defines all the operation performed to create and enqueue the flaky task
type FlakyTaskBuilder interface {
	// CreateFlakyTask creates flaky task and insert them in queue
	CreateFlakyTask(ctx context.Context, build *BuildCache, lastExecTask *Task, flakyConfig *FlakyConfig, orgID string) error
}
