package core

import (
	"context"
)

// TaskBuilder is helper interface for common function related to task creation
type TaskBuilder interface {
	// CreateTask creates and returns the task for execution and flaky
	// TODO : add support for discovery task
	CreateTask(ctx context.Context, taskID, buildID, orgID, repoID, subModule string, taskType TaskType, locatorConfig *InputLocatorConfig,
		tier Tier) (*Task, error)

	// CreateJob creates and return job object
	CreateJob(taskID, orgID string, status QueueStatus) *Job

	// EnqueueTaskAndJob creates task entry in db and enqueue respective jobs to queue
	EnqueueTaskAndJob(ctx context.Context, buildID, orgID string, tasks []*Task, jobs []*Job, taskType TaskType) error
}
