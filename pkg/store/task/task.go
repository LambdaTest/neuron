package task

import (
	"context"
	"fmt"
	"time"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/jmoiron/sqlx"
)

type taskStore struct {
	db      core.DB
	redisDB core.RedisDB
	logger  lumber.Logger
}

// New returns a new TaskStore.
func New(db core.DB, redisDB core.RedisDB, logger lumber.Logger) core.TaskStore {
	return &taskStore{db: db, redisDB: redisDB, logger: logger}
}

func (t *taskStore) Create(ctx context.Context, task ...*core.Task) error {
	return t.db.Execute(func(db *sqlx.DB) error {
		if _, err := db.NamedExecContext(ctx, insertQuery, task); err != nil {
			return err
		}
		return nil
	})
}

func (t *taskStore) CreateInTx(ctx context.Context, tx *sqlx.Tx, task ...*core.Task) error {
	if _, err := tx.NamedExecContext(ctx, insertQuery, task); err != nil {
		return err
	}
	return nil
}

func (t *taskStore) UpdateInTx(ctx context.Context, tx *sqlx.Tx, task *core.Task) error {
	_, err := tx.NamedExecContext(ctx, updateQuery, task)
	return err
}

func (t *taskStore) StopStaleTasksInTx(ctx context.Context, tx *sqlx.Tx, timeout time.Duration) (int64, error) {
	var updated int64
	query := fmt.Sprintf(markStaleStoppedQuery, timeout.Hours())
	res, err := tx.ExecContext(ctx, query, errs.GenericTaskTimeoutErrorRemark.Error())
	if err != nil {
		return 0, err
	}
	updated, err = res.RowsAffected()
	return updated, err
}

func (t *taskStore) UpdateByBuildID(ctx context.Context, status core.Status, taskType core.TaskType, remark, buildID string) error {
	return t.db.Execute(func(db *sqlx.DB) error {
		query := updateStatusByBuildQuery
		args := []interface{}{status, remark, buildID}
		if taskType != "" {
			query += ` AND type = ?`
			args = append(args, taskType)
		}
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			return err
		}
		return nil
	})
}

func (t *taskStore) UpdateByBuildIDInTx(ctx context.Context, tx *sqlx.Tx, status core.Status, remark, buildID string) error {
	if _, err := tx.ExecContext(ctx, updateStatusByBuildQuery, status, remark, buildID); err != nil {
		return err
	}
	return nil
}

func (t *taskStore) Find(ctx context.Context, taskID string) (*core.Task, error) {
	task := &core.Task{}
	return task, t.db.Execute(func(db *sqlx.DB) error {
		rows := db.QueryRowxContext(ctx, selectQuery, taskID)
		if err := rows.StructScan(task); err != nil {
			return err
		}
		return nil
	})
}

func (t *taskStore) FetchTask(ctx context.Context,
	buildID string,
	offset, limit int) ([]*core.Task, error) {
	tasks := make([]*core.Task, 0)
	return tasks, t.db.Execute(func(db *sqlx.DB) error {
		args := []interface{}{buildID, buildID, buildID, limit, offset}
		rows, err := db.QueryxContext(ctx, fetchTasksQuery, args...)
		if err != nil {
			return errs.SQLError(err)
		}
		defer rows.Close()

		for rows.Next() {
			task := new(core.Task)
			task.TestsMeta = new(core.ExecutionMeta)
			task.FlakyMeta = new(core.FlakyExecutionMetadata)
			if err := rows.Scan(&task.ID,
				&task.Created, &task.StartTime, &task.EndTime,
				&task.Status, &task.Type, &task.SubModule,
				&task.Remark, &task.TestsMeta.Total, &task.TestsMeta.TotalTestDuration,
				&task.TestsMeta.Passed, &task.TestsMeta.Failed, &task.TestsMeta.Skipped,
				&task.TestsMeta.Blocklisted, &task.TestsMeta.Quarantined, &task.TestsMeta.Aborted,
				&task.FlakyMeta.ImpactedTests, &task.FlakyMeta.FlakyTests, &task.FlakyMeta.Skipped,
				&task.FlakyMeta.Blocklisted, &task.FlakyMeta.Aborted,
			); err != nil {
				return errs.SQLError(err)
			}
			// TODO: This will be modified after matrix support
			task.Label = task.SubModule
			tasks = append(tasks, task)
		}
		return nil
	})
}

func (t *taskStore) FetchTaskHavingStatus(ctx context.Context,
	buildID string, status core.Status) ([]*core.Task, error) {
	tasks := make([]*core.Task, 0)
	return tasks, t.db.Execute(func(db *sqlx.DB) error {
		rows, err := db.QueryxContext(ctx, fetchTasksHavingStatusQuery, buildID, status)
		if err != nil {
			return errs.SQLError(err)
		}
		defer rows.Close()

		for rows.Next() {
			task := new(core.Task)
			if err := rows.StructScan(&task); err != nil {
				return errs.SQLError(err)
			}
			tasks = append(tasks, task)
		}
		return nil
	})
}

const insertQuery = `
INSERT
	INTO
	task(
		id,
		repo_id,
		build_id,
		status,
		test_locators,
		type,
		container_image,
		submodule
	)
VALUES (
	:id,
	:repo_id,
	:build_id,
	:status,
	:test_locators,
	:type,
	:container_image,
	:submodule
)
`

const updateQuery = `
UPDATE
	task
SET
	status = :status,
	remark = :remark,
	updated_at = :updated_at,
	start_time = :start_time,
	end_time = :end_time
WHERE
	id =:id
`

const updateStatusByBuildQuery = `
UPDATE
	task
SET
	status = ?,
	remark = ?,
	updated_at = NOW()
WHERE
	build_id=?
`

const selectQuery = `
SELECT
	id,
	repo_id,
	build_id,
	status,
	test_locators,
	type,
	created_at,
	updated_at,
	container_image,
	submodule,
	remark
FROM
	task
WHERE
	id = ?
`

const markStaleStoppedQuery = `
UPDATE
	task
SET
	status = 'error',
	remark = ?,
	updated_at = NOW()
WHERE
	DATE_ADD(created_at, INTERVAL %f HOUR) <= NOW()
	AND status IN ('initiating', 'running');
`

const fetchTasksQuery = `
WITH flaky_meta AS (
	SELECT
		ANY_VALUE(fte.build_id) AS build_id,
		fte.task_id AS task_id,
		COUNT(distinct fte.test_id) AS impacted_tests,
		COUNT(CASE WHEN fte.status = 'flaky' THEN fte.id end) flaky_tests,
		COUNT(CASE WHEN fte.status = 'skipped' THEN fte.id end) skipped,
		COUNT(CASE WHEN fte.status = 'blocklisted' THEN fte.id end) blocklisted,
		COUNT(CASE WHEN fte.status = 'aborted' THEN fte.id end) aborted
	FROM 
		flaky_test_execution fte
	WHERE
		fte.build_id = ? 
	GROUP BY fte.task_id 
	),
	execution_meta AS (
	SELECT
		ANY_VALUE(te.build_id) build_id,
		te.task_id, 
		COUNT(te.id) total_tests_executed,
		SUM(IFNULL(te.duration, 0)) total_duration,
		COUNT(CASE WHEN te.status = 'passed' THEN te.id end) passed,
		COUNT(CASE WHEN te.status = 'failed' THEN te.id end) failed,
		COUNT(CASE WHEN te.status = 'skipped' THEN te.id end) skipped,
		COUNT(CASE WHEN te.status = 'blocklisted' THEN te.id end) blocklisted,
		COUNT(CASE WHEN te.status = 'quarantined' THEN te.id end) quarantined,
		COUNT(CASE WHEN te.status = 'aborted' THEN te.id end) aborted
	FROM
		test_execution te
	WHERE
		te.build_id = ?
	GROUP BY
		te.task_id
	)
	SELECT
		ta.id,
		ta.created_at, 
		ta.start_time,
		ta.end_time,
		ta.status,
		ta.type,
		ta.submodule,
		ta.remark,
		COALESCE(em.total_tests_executed, 0) total_tests_executed,
		COALESCE(em.total_duration, 0) total_duration,
		COALESCE(em.passed, 0) passed,
		COALESCE(em.failed, 0) failed,
		COALESCE(em.skipped, 0) skipped,
		COALESCE(em.blocklisted, 0) blocklisted,
		COALESCE(em.quarantined, 0) quarantined,
		COALESCE(em.aborted, 0) aborted,
		COALESCE(fm.impacted_tests, 0) impacted_tests,
		COALESCE(fm.flaky_tests, 0) flaky_tests,
		COALESCE(fm.skipped, 0) flaky_skipped,
		COALESCE(fm.blocklisted, 0) flaky_blocklisted,
		COALESCE(fm.aborted, 0) flaky_aborted 
	FROM
		task ta
		LEFT JOIN flaky_meta fm ON ta.build_id = fm.build_id AND ta.id = fm.task_id
		LEFT JOIN execution_meta em ON ta.build_id = em.build_id AND ta.id = em.task_id 
WHERE
	ta.build_id = ?
ORDER BY ta.created_at LIMIT ? OFFSET ?
`

const fetchTasksHavingStatusQuery = `
SELECT
	t.id, 
	t.build_id,
	t.status
FROM
	task t
WHERE
	t.build_id = ?
	AND t.status = ?
`
