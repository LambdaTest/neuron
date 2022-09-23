package eventqueue

import (
	"context"
	"fmt"
	"time"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/jmoiron/sqlx"
)

type eventQueueStore struct {
	db     core.DB
	logger lumber.Logger
}

// New returns a new EventQueueStore.
func New(db core.DB, logger lumber.Logger) core.EventQueueStore {
	return &eventQueueStore{db: db, logger: logger}
}

func (e *eventQueueStore) CreateInTx(ctx context.Context, tx *sqlx.Tx, events ...*core.QueueableEvent) error {
	if _, err := tx.NamedExecContext(ctx, insertQuery, events); err != nil {
		return errs.SQLError(err)
	}
	return nil
}

func (e *eventQueueStore) UpdateBulkByIDInTx(ctx context.Context,
	tx *sqlx.Tx,
	status core.QueueStatus,
	eventIDs ...string) (int64, error) {
	arg := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
		"id":         eventIDs,
	}
	query, args, err := sqlx.Named(updateBulkByIDQuery, arg)
	if err != nil {
		return 0, errs.SQLError(err)
	}
	query, args, err = sqlx.In(query, args...)
	if err != nil {
		return 0, errs.SQLError(err)
	}
	query = tx.Rebind(query)
	res, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (e *eventQueueStore) UpdateBulkInTx(ctx context.Context, tx *sqlx.Tx, events ...*core.QueueableEvent) error {
	_, err := tx.NamedExecContext(ctx, insertQuery+onDuplicateKey, events)
	if err != nil {
		return errs.SQLError(err)
	}
	return nil
}

func (e *eventQueueStore) UpdateInTx(ctx context.Context, tx *sqlx.Tx, event *core.QueueableEvent) error {
	_, err := tx.NamedExecContext(ctx, updateQuery, event)
	if err != nil {
		return errs.SQLError(err)
	}
	return nil
}

func (e *eventQueueStore) FindAndUpdateInTx(ctx context.Context,
	tx *sqlx.Tx,
	buildID string,
	eventType core.QueueableEventType,
	refEntityType core.RefEntityType,
	queueStatus core.QueueStatus) (bool, error) {
	event, err := e.FindByRefEntityIDInTx(ctx, tx, eventType, buildID, refEntityType, true)
	if err != nil {
		return false, err
	}
	// if job alreadyFinished then we will decide if we want to decrement count in redis or not.
	alreadyFinished := utils.QueueJobFinshed(event.Status)
	event.Status = queueStatus
	event.Updated = time.Now()
	if err := e.UpdateInTx(ctx, tx, event); err != nil {
		return false, err
	}
	return alreadyFinished, nil
}

func (e *eventQueueStore) FindByRefEntityIDInTx(ctx context.Context,
	tx *sqlx.Tx,
	eventType core.QueueableEventType,
	refEntityID string,
	refEntityType core.RefEntityType,
	readLock bool) (*core.QueueableEvent, error) {
	query := selectQuery
	if readLock {
		query += forUpdate
	}
	args := []interface{}{refEntityID, refEntityType, eventType}
	row := tx.QueryRowxContext(ctx, query, args...)
	event := new(core.QueueableEvent)
	if err := row.StructScan(event); err != nil {
		return nil, errs.SQLError(err)
	}
	return event, nil
}

func (e *eventQueueStore) FindByOrgIDAndStatusInTx(ctx context.Context,
	tx *sqlx.Tx,
	orgID string,
	status core.QueueStatus,
	eventType core.QueueableEventType,
	readLock bool,
	limit int) ([]*core.QueueableEvent, error) {
	query := findByOrgIDStatusQuery
	if readLock {
		query += forUpdate
	}
	args := []interface{}{orgID, status, eventType, limit}
	rows, err := tx.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, errs.SQLError(err)
	}
	defer rows.Close()
	events := make([]*core.QueueableEvent, 0)
	for rows.Next() {
		event := new(core.QueueableEvent)
		if err := rows.StructScan(event); err != nil {
			return nil, errs.SQLError(err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, errs.SQLError(err)
	}
	if len(events) == 0 {
		return nil, errs.ErrRowsNotFound
	}
	return events, nil
}

func (e *eventQueueStore) FetchStatusCountInTx(ctx context.Context,
	tx *sqlx.Tx,
	orgID string,
	eventType core.QueueableEventType,
	readLock bool) (readyCount, processingCount int64, err error) {
	query := selectStatusCountQuery
	if readLock {
		query += forUpdate
	}
	row := tx.QueryRowxContext(ctx, query, orgID, eventType)
	err = row.Scan(&readyCount, &processingCount)
	return readyCount, processingCount, errs.SQLError(err)
}

func (e *eventQueueStore) FindOrgs(ctx context.Context,
	status core.QueueStatus,
	eventType core.QueueableEventType) ([]string, error) {
	orgIDs := make([]string, 0)
	return orgIDs, e.db.Execute(func(db *sqlx.DB) error {
		args := []interface{}{status, eventType}
		rows, err := db.QueryxContext(ctx, findOrgsQuery, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var org string
			if err = rows.Scan(&org); err != nil {
				return errs.SQLError(err)
			}
			orgIDs = append(orgIDs, org)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		return nil
	})
}

func (e *eventQueueStore) StopStaleEventsInTx(ctx context.Context, tx *sqlx.Tx, timeout time.Duration) (int64, error) {
	var updated int64
	query := fmt.Sprintf(markStaleStoppedQuery, timeout.Hours())
	res, err := tx.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}
	updated, err = res.RowsAffected()
	return updated, err
}

func (e *eventQueueStore) FetchStaleEventCountInTx(ctx context.Context, tx *sqlx.Tx,
	timeout time.Duration) ([]*core.StaleEvents, error) {
	staleEvents := make([]*core.StaleEvents, 0)
	query := fmt.Sprintf(selectStaleEventQuery, timeout.Hours())
	rows, err := tx.QueryxContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		event := new(core.StaleEvents)
		if err = rows.StructScan(event); err != nil {
			return nil, errs.SQLError(err)
		}
		staleEvents = append(staleEvents, event)
	}
	if err := rows.Err(); err != nil {
		return nil, errs.SQLError(err)
	}
	return staleEvents, nil
}

const selectStaleEventQuery = `
WITH stale_events AS (SELECT 
    id,
    org_id,
    status,
    event_type 
 FROM event_queue
 WHERE
 status IN ('ready', 'processing') AND
 DATE_ADD(created_at, INTERVAL %f HOUR) <= NOW()
)
SELECT
    org_id,
	COUNT(CASE WHEN event_type='task' AND status = 'ready' THEN id END) ready_tasks,
	COUNT(CASE WHEN event_type='task' AND status = 'processing' THEN id END) processing_tasks
FROM
	stale_events
GROUP BY org_id
`

const markStaleStoppedQuery = `
UPDATE
	event_queue
SET
	status = 'failed',
	updated_at = NOW()
WHERE
	DATE_ADD(created_at, INTERVAL %f HOUR) <= NOW()
	AND status IN ('processing', 'ready');
`
const forUpdate = ` FOR UPDATE`

const insertQuery = `
INSERT
	INTO
	event_queue(
		id,
		status,
		org_id,
		event_type,
		ref_entity_id,
		ref_entity_type
	)
VALUES (
	:id,
	:status,
	:org_id,
	:event_type,
	:ref_entity_id,
	:ref_entity_type
)
`

const updateQuery = `
UPDATE
	event_queue
SET
	status = :status,
	updated_at = :updated_at
WHERE
	id = :id
`
const onDuplicateKey = `
ON DUPLICATE KEY
UPDATE
	status = VALUES(status),
	updated_at = VALUES(updated_at)
`

const findByOrgIDStatusQuery = `
SELECT
	id,
	status,
	org_id,
	event_type,
	ref_entity_id,
	ref_entity_type
FROM
	event_queue
WHERE
	org_id = ?
	AND status = ?
	AND event_type = ?
ORDER BY
	created_at ASC
LIMIT ?
`

const findOrgsQuery = `
SELECT
	org_id
FROM
	event_queue
WHERE
	status = ?
	AND event_type = ?
GROUP BY
	org_id
`

const selectStatusCountQuery = `
SELECT
	COUNT(CASE WHEN status = 'ready' THEN id END)ready,
	COUNT(CASE WHEN status = 'processing' THEN id END)processing
FROM
	event_queue
WHERE
	org_id = ?
	AND event_type = ?
`

const selectQuery = `
SELECT
	id,
	status,
	org_id,
	event_type,
	ref_entity_id,
	ref_entity_type
FROM
	event_queue
WHERE
	ref_entity_id = ?
	AND ref_entity_type = ?
	AND event_type = ?
`

const updateBulkByIDQuery = `
UPDATE
	event_queue
SET
	status = :status,
	updated_at = :updated_at
WHERE
	id IN (:id)
	AND status != :status
`
