package flakyteststore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/gocraft/dbr"
	"github.com/gocraft/dbr/dialect"
	"github.com/jmoiron/sqlx"
)

type flakyTestStore struct {
	db     core.DB
	logger lumber.Logger
}

const insertQueryChunkSize = 1000

// New returns a new FlakyTestStore.
func New(db core.DB, logger lumber.Logger) core.FlakyTestStore {
	return &flakyTestStore{db: db, logger: logger}
}

func (fts *flakyTestStore) CreateInTx(ctx context.Context, tx *sqlx.Tx, testResults []*core.FlakyTestExecution) error {
	return utils.Chunk(insertQueryChunkSize, len(testResults), func(start int, end int) error {
		args := []interface{}{}
		placeholderGrps := []string{}
		for _, result := range testResults[start:end] {
			placeholderGrps = append(placeholderGrps, "(?,?,?,?,?,?,?)")
			args = append(args, result.ID, result.TestID, result.AlgoName, result.ExecInfo, result.Status, result.BuildID, result.TaskID)
		}
		interpolatedQuery, errI := dbr.InterpolateForDialect(fmt.Sprintf(insertQuery, strings.Join(placeholderGrps, ",")), args, dialect.MySQL)
		if errI != nil {
			return errs.SQLError(errI)
		}
		if _, err := tx.ExecContext(ctx, interpolatedQuery); err != nil {
			return errs.SQLError(err)
		}
		return nil
	})
}

func (fts *flakyTestStore) FindTests(ctx context.Context, buildID, taskID string,
	statusFilter, executionStatusFilter, searchText string, offset, limit int) ([]*core.FlakyTestExecution, error) {
	tests := make([]*core.FlakyTestExecution, 0)
	return tests, fts.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"build_id":         buildID,
			"status":           statusFilter,
			"execution_status": executionStatusFilter,
			"limit":            limit,
			"offset":           offset,
			"search_text":      "%" + searchText + "%",
			"task_id":          taskID,
		}

		statusWhere := ""
		executionStatusWhere := ""
		searchWhere := ""
		taskWhere := ""
		if statusFilter != "" {
			statusWhere = `AND fte.status = :status`
		}
		if executionStatusFilter != "" {
			statusWhere = `AND te.status = :execution_status`
		}
		if searchText != "" {
			searchWhere = `AND t.name LIKE :search_text`
		}
		if taskID != "" {
			taskWhere = `AND fte.task_id = :task_id`
		}
		query := fmt.Sprintf(findTestQuery, statusWhere, executionStatusWhere, searchWhere, taskWhere)
		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}

		defer rows.Close()
		for rows.Next() {
			t := new(core.FlakyTestExecution)
			execInfo := &core.AggregatedFlakyTestExecInfo{}
			var execInfoRaw sql.NullString
			var status sql.NullString
			var locator string
			if err = rows.Scan(&t.TestID,
				&locator,
				&t.TestName,
				&t.TestSuiteID,
				&t.TestSuiteName,
				&execInfoRaw,
				&status,
				&t.ExecutionStatus); err != nil {
				return errs.SQLError(err)
			}

			if execInfoRaw.Valid {
				if err := json.Unmarshal([]byte(execInfoRaw.String), execInfo); err != nil {
					return err
				}
			}

			if status.Valid {
				t.Status = core.TestExecutionStatus(status.String)
			}

			t.FileName = utils.GetFileNameFromTestLocator(locator)
			t.ExecutionInfo = execInfo
			t.TestResults = execInfo.Results
			if execInfo.NumberOfExecutions > 0 {
				t.FlakyRate = float32(execInfo.TotalTransitions) / float32(execInfo.NumberOfExecutions)
			}
			tests = append(tests, t)
		}

		if len(tests) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
}

func (fts *flakyTestStore) MarkTestsToStableInTx(ctx context.Context,
	tx *sqlx.Tx, buildID string) (int64, error) {
	args := map[string]interface{}{
		"build_id": buildID,
	}
	result, err := tx.NamedExecContext(ctx, markTestsToStableQuery, args)
	if err != nil {
		return 0, errs.SQLError(err)
	}
	rowCount, err := result.RowsAffected()
	if err != nil {
		return 0, errs.SQLError(err)
	}
	return rowCount, nil
}

func (fts *flakyTestStore) FindTestsInJob(ctx context.Context, repoName, orgID, branchName string,
	startDate, endDate time.Time) ([]*core.FlakyTestExecution, error) {
	tests := make([]*core.FlakyTestExecution, 0)
	return tests, fts.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo":       repoName,
			"org_id":     orgID,
			"branch":     branchName,
			"start_date": startDate,
			"end_date":   endDate,
		}
		branchWhere := ""
		if branchName != "" {
			branchWhere = branchWhereClause
		}
		query := fmt.Sprintf(flakyTestsInJobQuery, branchWhere)
		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}
		defer rows.Close()
		for rows.Next() {
			t := new(core.FlakyTestExecution)
			t.TestMeta = new(core.FlakyExecutionMetadata)
			if err = rows.Scan(&t.BuildID,
				&t.TestMeta.ImpactedTests,
				&t.TestMeta.FlakyTests,
				&t.TestMeta.NonFlakyTests,
				&t.TestMeta.Aborted,
				&t.TestMeta.Blocklisted,
				&t.TestMeta.Skipped,
				&t.TestMeta.Quarantined,
				&t.CreatedAt); err != nil {
				return errs.SQLError(err)
			}
			tests = append(tests, t)
		}

		if len(tests) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
}

func (fts *flakyTestStore) ListFlakyTests(ctx context.Context, repoName, orgID, branchName string,
	startDate, endDate time.Time, offset, limit int) ([]*core.FlakyTestExecution, error) {
	tests := make([]*core.FlakyTestExecution, 0)
	return tests, fts.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo":       repoName,
			"org_id":     orgID,
			"branch":     branchName,
			"start_date": startDate,
			"end_date":   endDate,
			"limit":      limit,
			"offset":     offset,
		}
		branchWhere := ""
		if branchName != "" {
			branchWhere = branchWhereClause
		}
		query := fmt.Sprintf(flakyTestsListQuery, branchWhere, branchWhere, branchWhere)
		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}
		defer rows.Close()
		for rows.Next() {
			t := new(core.FlakyTestExecution)
			t.TestMeta = new(core.FlakyExecutionMetadata)
			var transition, totalExecution int
			if err = rows.Scan(&t.TestID,
				&t.TestName,
				&t.TestSuiteName,
				&t.TestSuiteID,
				&t.TestMeta.JobsCount,
				&t.TestMeta.LastFlakeID,
				&t.TestMeta.LastFlakeTime,
				&t.TestMeta.LastFlakeCommitID,
				&t.TestMeta.FirstFlakeID,
				&t.TestMeta.FirstFlakeTime,
				&t.TestMeta.FirstFlakeCommitID,
				&t.Status,
				&transition,
				&totalExecution); err != nil {
				return errs.SQLError(err)
			}
			if totalExecution > 0 {
				t.FlakyRate = float32(transition) / float32(totalExecution)
			}
			tests = append(tests, t)
		}

		if len(tests) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
}

const branchWhereClause = " AND b.branch_name = :branch "

const insertQuery = `
INSERT
	INTO
	flaky_test_execution(
		id,
		test_id,
		algo_name,
		exec_info,
		status,
		build_id,
		task_id
	)
VALUES %s
`

const findTestQuery = `
SELECT
	t.id as test_id,
	t.test_locator,
	t.name,
	ts.id as test_suite_id ,
	ts.name as test_suite_name,
	fte.exec_info,
	fte.status,
	te.status as execution_status
FROM
	test_execution te
LEFT JOIN build b ON 
    b.id = te.build_id
LEFT JOIN test t ON 
	t.id = te.test_id 
LEFT JOIN test_suite ts ON 
	t.test_suite_id =ts.id 
LEFT JOIN  flaky_test_execution fte ON 
	fte.test_id  = te.test_id AND fte.build_id = te.build_id 
WHERE b.id = :build_id
	%s
	%s
	%s
	%s
ORDER BY fte.test_id DESC, fte.created_at DESC 
LIMIT :limit OFFSET :offset
`

const markTestsToStableQuery = `
WITH stable_tests AS (
SELECT 
	fte.test_id,
	fte.status,
	b.repo_id,
	b.branch_name 
FROM
	flaky_test_execution fte
JOIN build b ON
	b.id = fte.build_id 
WHERE
	b.id = :build_id
	AND fte.status = 'nonflaky'
), quarantine_tests AS (SELECT 
	b.id
FROM
	block_tests b
JOIN stable_tests st ON
	st.repo_id = b.repo_id 
	AND st.test_id = b.test_id
	AND st.branch_name = b.branch
WHERE
	b.status = 'quarantined'
)
UPDATE
   block_tests b,
   quarantine_tests qt
SET 
   b.status = 'nonflaky' 
WHERE 
    qt.id = b.id
`
const flakyTestsInJobQuery = `
SELECT
	fte.build_id build_id,
	COUNT(DISTINCT fte.test_id) total_tests,
	COUNT(DISTINCT CASE WHEN fte.status = 'flaky' THEN fte.test_id END) flaky_tests,
	COUNT(DISTINCT CASE WHEN fte.status = 'nonflaky' THEN fte.test_id END) nonflaky_tests,
	COUNT(DISTINCT CASE WHEN fte.status = 'aborted' THEN fte.test_id END) aborted_tests,
	COUNT(DISTINCT CASE WHEN fte.status = 'blocklisted' THEN fte.test_id END) blocklisted_tests,
	COUNT(DISTINCT CASE WHEN fte.status = 'skipped' THEN fte.test_id END) skipped_tests,
	COUNT(DISTINCT CASE WHEN fte.status = 'quarantined' THEN fte.test_id END) quarantined_tests,
	MAX(b.created_at) created_at 
FROM
	flaky_test_execution fte
JOIN build b ON
	b.id = fte.build_id
JOIN repositories r ON
	r.id = b.repo_id
WHERE
	r.name = :repo
	AND r.org_id = :org_id
	%s
	AND b.created_at >= :start_date
	AND b.created_at <= :end_date
GROUP BY build_id ORDER BY created_at DESC `

const flakyTestsListQuery = `
WITH flaky_job_ranked AS (
	SELECT 
		fte.test_id,
		fte.created_at,
		fte.build_id,
		b.commit_id,
		ROW_NUMBER() OVER (PARTITION BY fte.test_id
	ORDER BY
			fte.updated_at DESC ) rn,
		ROW_NUMBER() OVER (PARTITION BY fte.test_id
	ORDER BY
			fte.updated_at ) rn_desc
	FROM
		flaky_test_execution fte
	JOIN build b ON
		b.id = fte.build_id
	JOIN repositories r ON
		r.id = b.repo_id
	WHERE
		r.name = :repo
		AND r.org_id = :org_id
		AND fte.status = "flaky"
		AND b.created_at >= :start_date
		AND b.created_at <= :end_date
		%s
	),
	latest_flaky_job AS (
	SELECT
		*
	FROM
		flaky_job_ranked
	WHERE
		rn = 1
	),
	first_flaky_job AS (
	SELECT
		*
	FROM
		flaky_job_ranked
	WHERE
		rn_desc = 1
	),
	latest_status_ranked AS (
	SELECT 
		fte.test_id,
		fte.created_at,
		fte.build_id,
		fte.status,
		ROW_NUMBER() OVER (PARTITION BY fte.test_id
	ORDER BY
			fte.updated_at DESC ) rn
	FROM
		flaky_test_execution fte
	JOIN build b ON
		b.id = fte.build_id
	JOIN repositories r ON
		r.id = b.repo_id
	WHERE
		r.name = :repo
		AND r.org_id = :org_id
		AND b.created_at >= :start_date
		AND b.created_at <= :end_date
		%s
	),
	latest_status AS (
	SELECT
		*
	FROM
		latest_status_ranked
	WHERE
		rn = 1
	)
	SELECT
		fte.test_id,
		t.name test_name,
		ts.name suite_name,
		ts.id suite_id,
		COUNT(fte.build_id) occurrences,
		ANY_VALUE(lfj.build_id),
		ANY_VALUE(lfj.created_at),
		ANY_VALUE(lfj.commit_id),
		ANY_VALUE(ffj.build_id),
		ANY_VALUE(ffj.created_at),
		ANY_VALUE(ffj.commit_id),
		ANY_VALUE(ls.status),
		SUM(JSON_EXTRACT(fte.exec_info, "$.total_transitions")) transition,
		SUM(JSON_EXTRACT(fte.exec_info, "$.number_of_executions")) total_execution
	FROM
		flaky_test_execution fte
	JOIN build b ON
		b.id = fte.build_id
	JOIN repositories r ON
		r.id = b.repo_id
	JOIN test t ON
		t.id = fte.test_id
	LEFT JOIN test_suite ts ON
		ts.id = t.test_suite_id
	JOIN first_flaky_job ffj ON
		ffj.test_id = fte.test_id
	JOIN latest_flaky_job lfj ON 
		lfj.test_id = fte.test_id
	JOIN latest_status ls ON
		ls.test_id = fte.test_id
	WHERE
		r.name = :repo
		AND r.org_id = :org_id
		AND fte.status = "flaky"
		AND b.created_at >= :start_date
		AND b.created_at <= :end_date
		%s
	GROUP BY
		fte.test_id
	ORDER BY
		fte.test_id DESC
	LIMIT :limit
	OFFSET :offset
`
