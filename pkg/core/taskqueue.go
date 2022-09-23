package core

import (
	"context"
	"time"
)

// Job represents the item queued in task queue.
type Job struct {
	ID      string      `json:"id"`
	Status  QueueStatus `json:"status"`
	OrgID   string      `json:"org_id"`
	TaskID  string      `json:"task_id"`
	Created time.Time   `json:"created_at"`
	Updated time.Time   `json:"updated_at"`
}

// ToQueueableEvent is utility function to convert a Job to QueueableEvent
func (j *Job) ToQueueableEvent() *QueueableEvent {
	return &QueueableEvent{
		ID:            j.ID,
		Status:        j.Status,
		OrgID:         j.OrgID,
		Type:          TaskEvent,
		RefEntityID:   j.TaskID,
		RefEntityType: RefEntityTaskType,
		Created:       j.Created,
		Updated:       j.Updated,
	}
}

// FromQueueableEventToJob is utility function to convert a QueueableEvent to Job
func FromQueueableEventToJob(event *QueueableEvent) *Job {
	return &Job{
		ID:      event.ID,
		Status:  event.Status,
		OrgID:   event.OrgID,
		TaskID:  event.RefEntityID,
		Created: event.Created,
		Updated: event.Updated,
	}
}

// TaskQueueScheduler will schedule tasks which are in ready state after startup.
type TaskQueueScheduler interface {
	//  Run starts the scheduler on startup.
	Run(ctx context.Context)
}

// TaskQueueManager manages the task queue.
type TaskQueueManager interface {
	// EnqueueTasks inserts tasks in queue.
	EnqueueTasks(orgID, buildID string, jobs ...*Job) error
	// DequeueTasks Dequeue tasks for the orgID.
	DequeueTasks(orgID string) error
	// Close closes the queue.
	Close() error
}

// TaskQueueStore represents the task_queue store operations.
type TaskQueueStore interface {
	// Create create tasks in tasksqueue store.
	Create(ctx context.Context, orgID, buildID string, tasks ...*Job) error
	// FindAndUpdateTasks finds and updates the status of tasks in the task queue store and returns the messages.
	FindAndUpdateTasks(ctx context.Context, orgID string, limit int) ([]*Job, error)
	// UpdateTask updates the task in queue, updates credit usage, updates task table and decrements the count in redis.
	UpdateTask(ctx context.Context, task *Task, orgID string) error
	// MarkError marks all tasks and build as error and tasks failed in task queue.
	MarkError(ctx context.Context, buildID, orgID, remark string, jobs []*Job) error
}

// TaskQueueUtils has the common utilities for task queue.
type TaskQueueUtils interface {
	// MarkTaskToStatus marks the tasks as error.
	MarkTaskToStatus(task *Task, orgID string, status Status)
}
