package core

import (
	"context"
)

const (
	// JobCompletedMsg stores the msg for already stopped job
	JobCompletedMsg = "job already completed"
	// JobAbortedByUser stores the remark to use while updating task because of abort build API
	JobAbortedByUser = "aborted by user"
)

// BuildAbortProducer represents the build abort producer.
type BuildAbortProducer interface {
	// Enqueue inserts payload in queue.
	Enqueue(buildID, orgID string) error
	// Close closes the queue producer.
	Close() error
}

// BuildAbortConsumer represents the  build abort consumer.
type BuildAbortConsumer interface {
	// Run runs the queue consumer in background.
	Run(ctx context.Context)
	// Close closes the queue consumer
	Close() error
}

// BuildAbortService implements the build abort service
type BuildAbortService interface {
	// AbortBuild aborts the running and initiating tasks
	AbortBuild(ctx context.Context, buildID, repoID, orgID string) (string, error)
}
