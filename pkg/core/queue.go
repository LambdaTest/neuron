package core

import "context"

// QueueStatus represents the status of tasks in queue.
type QueueStatus string

// List of task queue statuses
const (
	Ready      QueueStatus = "ready"
	Processing QueueStatus = "processing"
	Completed  QueueStatus = "completed"
	Failed     QueueStatus = "failed"
	Aborted    QueueStatus = "aborted"
)

// QueueProducer represents the queue producer.
type QueueProducer interface {
	// Enqueue inserts payload in queue.
	Enqueue(payload interface{}) error
	// Close closes the queue producer.
	Close() error
}

// QueueConsumer represents the queue consumer.
type QueueConsumer interface {
	// Run runs the queue consumer in background.
	Run(ctx context.Context)
	// Close closes the queue consumer
	Close() error
}
