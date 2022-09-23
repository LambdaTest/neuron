package core

import (
	"context"
)

// TaskUpdateManager defines utility functions for updating task(s)
type TaskUpdateManager interface {
	// TaskUpdate updates the task and build as per provided status
	TaskUpdate(ctx context.Context, task *Task, orgID string) error
	// UpdateAllTasksForBuild fetches and update all task for a build
	UpdateAllTasksForBuild(ctx context.Context, remark, buildID, orgID string) error
}
