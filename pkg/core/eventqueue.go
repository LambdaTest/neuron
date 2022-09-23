package core

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

// QueueableEventType represents the type of queueable event
type QueueableEventType string

// QueableEventType values exhaustive set
const (
	TaskEvent QueueableEventType = "task"
)

// RefEntityType represents the type of entity being queued as a queueable event
type RefEntityType string

// RefEntityType values exhaustive set
const (
	RefEntityTaskType RefEntityType = "task"
)

// QueueableEvent represents a singular event that can be DB backed queued
type QueueableEvent struct {
	ID            string             `db:"id"`
	Status        QueueStatus        `db:"status"`
	OrgID         string             `db:"org_id"`
	Type          QueueableEventType `db:"event_type" validate:"omitempty,oneof=task"`
	RefEntityID   string             `db:"ref_entity_id"`
	RefEntityType RefEntityType      `db:"ref_entity_type" validate:"omitempty,oneof=task"`
	Created       time.Time          `db:"created_at"`
	Updated       time.Time          `db:"updated_at"`
}

// StaleEvents stores org wise stale event counts
type StaleEvents struct {
	OrgID           string `json:"org_id" db:"org_id"`
	ReadyTasks      int64  `json:"ready_tasks" db:"ready_tasks"`
	ProcessingTasks int64  `json:"processing_tasks" db:"processing_tasks"`
}

// EventQueueStore defines datastore operation for working with event_queue data-store.
type EventQueueStore interface {
	// CreateInTx creates events in event_queue store.
	CreateInTx(ctx context.Context, tx *sqlx.Tx, events ...*QueueableEvent) error
	// UpdateBulkInTx updates the given events in event_queue store
	UpdateBulkInTx(ctx context.Context, tx *sqlx.Tx, events ...*QueueableEvent) error
	// UpdateInTx updates the event in event_queue store and executes the query within the specified transaction.
	UpdateInTx(ctx context.Context, tx *sqlx.Tx, event *QueueableEvent) error
	// FindAndUpdateInTx find and update the  event in event_queue store and executes the query within the specified transaction.
	FindAndUpdateInTx(ctx context.Context, tx *sqlx.Tx, buildID string, eventType QueueableEventType,
		refEntityType RefEntityType, queueStatus QueueStatus) (bool, error)
	// UpdateBulkByIDInTx updates the status of events in event_queue store by ID and returns the number of rows affected
	UpdateBulkByIDInTx(ctx context.Context, tx *sqlx.Tx, status QueueStatus, eventIDs ...string) (int64, error)
	// FindByRefEntityIDInTx finds event uniquely identified by refEntityID, refEntityType and eventType
	FindByRefEntityIDInTx(ctx context.Context, tx *sqlx.Tx,
		eventType QueueableEventType, refEntityID string, refEntityType RefEntityType, readLock bool) (*QueueableEvent, error)
	// FindByOrgIDAndStatusInTx finds events for a given orgID with given status with or without read lock on relevant rows
	FindByOrgIDAndStatusInTx(ctx context.Context, tx *sqlx.Tx,
		orgID string, status QueueStatus, eventType QueueableEventType, readLock bool, limit int) ([]*QueueableEvent, error)
	// FetchStatusCountInTx fetches the ready and processing counts for given orgID
	FetchStatusCountInTx(ctx context.Context, tx *sqlx.Tx,
		orgID string, eventType QueueableEventType, readLock bool) (readyCount, processingCount int64, err error)
	// FetchStaleEventCountInTx returns org wise ready and processing tasks for
	// task events after timeout within specified transaction
	FetchStaleEventCountInTx(ctx context.Context, tx *sqlx.Tx,
		timeout time.Duration) ([]*StaleEvents, error)
	// FindOrgs fetches all organizations that have events in event_queue table for a given status and particular event_type
	FindOrgs(ctx context.Context, status QueueStatus, eventType QueueableEventType) ([]string, error)
	// StopStaleEventsInTx marks the events as failed after timeout within specified transaction.
	StopStaleEventsInTx(ctx context.Context, tx *sqlx.Tx, timeout time.Duration) (int64, error)
}
