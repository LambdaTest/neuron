package testexecution

import (
	"context"
	"database/sql"
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

type testExecutionStore struct {
	db     core.DB
	logger lumber.Logger
}

const insertQueryChunkSize = 1000

// New returns a new TestExecutionStore.
func New(db core.DB, logger lumber.Logger) core.TestExecutionStore {
	return &testExecutionStore{db: db, logger: logger}
}

func (t *testExecutionStore) CreateInTx(ctx context.Context, tx *sqlx.Tx, testExecutionData []*core.TestExecution) error {
	return utils.Chunk(insertQueryChunkSize, len(testExecutionData), func(start int, end int) error {
		args := []interface{}{}
		placeholderGrps := []string{}
		for _, execData := range testExecutionData[start:end] {
			placeholderGrps = append(placeholderGrps, "(?,?,?,?,?,?,?,?,?,?)")
			args = append(args, execData.ID, execData.TestID, execData.BuildID, execData.TaskID, execData.Status, execData.CommitID,
				execData.StartTime, execData.EndTime, execData.Duration, execData.BlocklistSource)
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

func (t *testExecutionStore) FindByTestID(ctx context.Context,
	testID,
	repoName,
	orgID,
	branchName, statusFilter, searchText string,
	startDate, endDate time.Time,
	offset,
	limit int) ([]*core.TestExecution, error) {
	tests := make([]*core.TestExecution, 0)
	return tests, t.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"test_id":        testID,
			"repo":           repoName,
			"org_id":         orgID,
			"branch":         branchName,
			"status":         statusFilter,
			"search_text":    "%" + searchText + "%",
			"min_created_at": startDate,
			"max_created_at": endDate,
			"limit":          limit,
			"offset":         offset,
		}
		query := findByTestIDQuery
		branchWhere := ""
		statusWhere := ""
		searchWhere := ""
		if branchName != "" {
			branchWhere = branchCondition
		}
		if statusFilter != "" {
			statusWhere = `AND te.status = :status`
		}
		if searchText != "" {
			searchWhere = `AND ( te.commit_id LIKE :search_text OR te.build_id LIKE :search_text )`
		}
		query = fmt.Sprintf(query, branchWhere, statusWhere, searchWhere)
		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}
		defer rows.Close()
		for rows.Next() {
			t := new(core.TestExecution)
			if err = rows.Scan(&t.ID,
				&t.TestID,
				&t.CommitID,
				&t.CommitAuthor,
				&t.CommitMessage,
				&t.Status,
				&t.Duration,
				&t.Created,
				&t.StartTime,
				&t.EndTime,
				&t.BuildNum,
				&t.BuildID,
				&t.BuildTag,
				&t.TaskID); err != nil {
				return errs.SQLError(err)
			}
			tests = append(tests, t)
		}
		if rows.Err() != nil {
			return rows.Err()
		}

		if len(tests) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
}

func (t *testExecutionStore) FindStatus(ctx context.Context,
	testID,
	interval,
	repoName,
	orgID,
	branchName string,
	startDate, endDate time.Time) ([]*core.TestExecution, error) {
	tests := make([]*core.TestExecution, 0)
	return tests, t.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"test_id":        testID,
			"repo":           repoName,
			"org_id":         orgID,
			"branch":         branchName,
			"min_created_at": startDate,
			"max_created_at": endDate,
		}
		query := findStatusQuery
		branchJoin := ""
		branchWhere := ""
		if branchName != "" {
			branchJoin = `JOIN build b ON te.build_id = b.id`
			branchWhere = branchCondition
		}
		query = fmt.Sprintf(query, branchJoin, branchWhere)
		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}
		defer rows.Close()
		for rows.Next() {
			t := new(core.TestExecution)
			if err = rows.StructScan(t); err != nil {
				return errs.SQLError(err)
			}
			tests = append(tests, t)
		}
		return nil
	})
}

func (t *testExecutionStore) FindTimeByRunningAllTests(ctx context.Context,
	buildID,
	commitID, repoID string) (totalTests, totalTime int, err error) {
	err = t.db.Execute(func(db *sqlx.DB) error {
		args := []interface{}{buildID, repoID}
		query := findTimeByRunningAllTestsQuery
		if commitID == "" {
			query = fmt.Sprintf(query, "")
		} else {
			query = fmt.Sprintf(query, "AND b.commit_id = ?")
			args = append(args, commitID)
		}
		args = append(args, buildID)
		row := db.QueryRowxContext(ctx, query, args...)
		if errQ := row.Scan(&totalTests, &totalTime); errQ != nil {
			return errs.SQLError(errQ)
		}
		return nil
	})
	return totalTests, totalTime, err
}

func (t *testExecutionStore) FindExecutionTimeImpactedTests(ctx context.Context,
	commitID,
	buildID, repoID string) (totalImpactedTests, totalTime int, err error) {
	return totalImpactedTests, totalTime, t.db.Execute(func(db *sqlx.DB) error {
		args := []interface{}{buildID, repoID}
		query := findExecutionTimeImpactedTestsQuery
		if commitID == "" {
			query = fmt.Sprintf(query, "")
		} else {
			query = fmt.Sprintf(query, "AND b.commit_id = ?")
			args = append(args, commitID)
		}
		row := db.QueryRowxContext(ctx, query, args...)
		var dbTotalTime sql.NullInt32
		dbTotalImpactedTests := new(int)
		if err := row.Scan(&dbTotalTime, dbTotalImpactedTests); err != nil {
			return errs.SQLError(err)
		}
		if !dbTotalTime.Valid {
			return errs.ErrRowsNotFound
		}
		totalImpactedTests = *dbTotalImpactedTests
		totalTime = int(dbTotalTime.Int32)

		return nil
	})
}

func (t *testExecutionStore) FindRepoTestStatusesFailed(ctx context.Context, repoName, orgID, branchName string,
	startDate, endDate time.Time, limit int) ([]*core.Test, error) {
	tests := make([]*core.Test, 0, limit)
	return tests, t.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo":           repoName,
			"org_id":         orgID,
			"branch":         branchName,
			"min_created_at": startDate,
			"max_created_at": endDate,
			"limit":          limit,
		}
		query := findRepoTestStatusFailedQuery
		if branchName != "" {
			branchBuildJoin := " JOIN build b ON b.id = te.build_id "
			branchBuildWhere := " AND b.branch_name = :branch "
			query = fmt.Sprintf(query, branchBuildJoin, branchBuildWhere, branchBuildJoin, branchBuildWhere)
		} else {
			query = fmt.Sprintf(query, "", "", "", "")
		}
		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}
		defer rows.Close()
		for rows.Next() {
			t := new(core.Test)
			t.Execution = new(core.TestExecution)
			t.Meta = new(core.ExecutionMeta)
			if err = rows.Scan(&t.ID,
				&t.DebutCommit,
				&t.Name,
				&t.Execution.Duration,
				&t.Execution.Created,
				&t.Meta.Failed); err != nil {
				return errs.SQLError(err)
			}
			tests = append(tests, t)
		}

		if rows.Err() != nil {
			return rows.Err()
		}

		if len(tests) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
}

func (t *testExecutionStore) FindLatestExecution(ctx context.Context, repoID string, testIDs []string) ([]*core.Test, error) {
	tests := make([]*core.Test, 0)
	return tests, t.db.Execute(func(db *sqlx.DB) error {
		initArgs := map[string]interface{}{
			"repo_id": repoID,
		}
		selectQuery := latestTestExecutionQuery

		if len(testIDs) > 0 {
			selectQuery = fmt.Sprintf(selectQuery, "AND t.id IN (:id)")
			initArgs["id"] = testIDs
		} else {
			selectQuery = fmt.Sprintf(selectQuery, "")
		}

		query, args, err := sqlx.Named(selectQuery, initArgs)
		if err != nil {
			t.logger.Errorf("failed to created named query, error %v", err)
			return errs.SQLError(err)
		}
		query, args, err = sqlx.In(query, args...)
		if err != nil {
			t.logger.Errorf("failed to created IN query, error %v", err)
			return errs.SQLError(err)
		}
		query = db.Rebind(query)
		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			return errs.SQLError(err)
		}
		defer rows.Close()
		for rows.Next() {
			t := new(core.Test)
			t.Execution = new(core.TestExecution)
			if err = rows.Scan(&t.ID,
				&t.TestLocator,
				&t.Execution.Status,
				&t.Execution.Duration,
			); err != nil {
				return errs.SQLError(err)
			}
			tests = append(tests, t)
		}

		if rows.Err() != nil {
			return rows.Err()
		}

		if len(tests) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
}

func (t *testExecutionStore) FindImpactedTestsByBuild(ctx context.Context, buildID string) (map[string][]*core.Test, error) {
	testMap := map[string][]*core.Test{}
	err := t.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"build_id": buildID,
		}
		rows, err := db.NamedQueryContext(ctx, findImpactedTestByBuildQuery, args)
		if err != nil {
			return errs.SQLError(err)
		}
		for rows.Next() {
			testObj := new(core.Test)
			if scanErr := rows.StructScan(testObj); scanErr != nil {
				return errs.SQLError(scanErr)
			}
			if _, ok := testMap[testObj.SubModule]; !ok {
				testMap[testObj.SubModule] = make([]*core.Test, 0)
			}
			testMap[testObj.SubModule] = append(testMap[testObj.SubModule], testObj)
		}
		return nil
	})
	return testMap, err
}

const branchCondition = `AND b.branch_name = :branch`

const findByTestIDQuery = `
SELECT
	te.id,
	te.test_id,
	te.commit_id,
	c.author_name,
	c.message,
	te.status,
	te.duration,
	te.created_at,
	te.start_time,
	te.end_time,
	b.build_num,
	b.id,
	b.tag,
	te.task_id
FROM
	test_execution te
JOIN test t ON
	t.id = te.test_id
JOIN repositories r ON
	r.id = t.repo_id
JOIN build b ON
	b.id = te.build_id
JOIN git_commits c ON
	c.commit_id = te.commit_id
	AND c.repo_id = r.id
WHERE
	te.test_id = :test_id
	AND r.name = :repo
	AND r.org_id = :org_id
	AND te.created_at >= :min_created_at
	AND te.created_at <= :max_created_at
	%s
	%s
	%s
ORDER BY te.updated_at DESC
LIMIT :limit OFFSET :offset
`

const findStatusQuery = `
SELECT
	te.status,
	te.duration,
	te.created_at,
	te.start_time,
	te.end_time
FROM
	test_execution te
JOIN test t ON
	t.id = te.test_id
JOIN repositories r ON
	r.id = t.repo_id
%s
WHERE
	r.name = :repo
	AND r.org_id = :org_id
	AND te.test_id = :test_id
	AND te.created_at >= :min_created_at
	AND te.created_at <= :max_created_at
	%s
`
const findExecutionTimeImpactedTestsQuery = `
SELECT
	SUM(te.duration) total_time,
	COUNT(1) total_impacted_tests
FROM
	test_execution te
JOIN build b
	ON te.build_id = b.id
WHERE
	te.build_id = ?
	AND b.repo_id = ?
	%s
`
const findTimeByRunningAllTestsQuery = `
WITH discovered_tests AS (
	SELECT
		tt.test_id,
		cd.repo_id
	FROM
		commit_discovery cd
	JOIN JSON_TABLE(
			cd.test_ids,
			'$[*]' COLUMNS(
				test_id varchar(32) PATH '$' ERROR ON
				ERROR
			)
		) tt
	JOIN build b ON
		b.commit_id = cd.commit_id
		AND b.repo_id = cd.repo_id
	WHERE
		b.id = ?
		AND cd.repo_id = ?
		%s
),
test_ids AS (
	SELECT
		DISTINCT test_id,
		repo_id
	FROM
		discovered_tests
),
test_runtimes AS (
	SELECT
		te2.test_id,
		MAX(te2.duration) duration
	FROM
		test_execution te2
	JOIN (
			SELECT
				te.test_id,
				MAX(te.updated_at) updated_at
			FROM
				test_ids ti
			JOIN test t ON
				ti.test_id = t.id
				AND ti.repo_id = t.repo_id
			JOIN test_execution te ON
				t.id = te.test_id
			WHERE
				te.updated_at <= (
					SELECT
						COALESCE(b.end_time, b.updated_at, b.created_at)
					FROM
						build b
					WHERE
						b.id = ?
				)
			GROUP BY
				te.test_id
		) temp ON
		temp.test_id = te2.test_id
		AND temp.updated_at = te2.updated_at
	GROUP BY
		te2.test_id
)
SELECT
	COUNT(DISTINCT dt.test_id) total_tests,
	IFNULL(SUM(tr.duration), 0)
FROM
	test_runtimes tr
	RIGHT JOIN discovered_tests dt ON dt.test_id = tr.test_id
`
const insertQuery = `
INSERT
	INTO
	test_execution(
		id,
		test_id,
		build_id,
		task_id,
		status,
		commit_id,
		start_time,
		end_time,
		duration,
		blocklist_source
	)
VALUES %s
`

const findRepoTestStatusFailedQuery = `
WITH ranked_executions AS(
	SELECT
		te.test_id,
		te.commit_id,
		ROW_NUMBER() OVER (PARTITION BY te.test_id
	ORDER BY
		te.updated_at DESC) rn
	FROM
		test_execution te
			%s
	JOIN test t ON
		t.id = te.test_id
	JOIN repositories r ON
		r.id = t.repo_id
	WHERE
		r.name = :repo
		AND r.org_id = :org_id
		AND te.created_at >= :min_created_at
		AND te.created_at <= :max_created_at
		%s
		AND te.status = "failed"
	),
	latest_commits_failed AS (
		SELECT 
			*
		FROM ranked_executions
		WHERE rn = 1
	)
	SELECT
		t.id,
		ANY_VALUE(lcf.commit_id),
		t.name,
		MAX(te.duration) duration,
		MAX(te.created_at) latest_execution,
		COUNT(DISTINCT CASE WHEN te.status = 'failed' THEN te.id END) failed
	FROM
		test_execution te
	JOIN test t ON
		t.id = te.test_id
	JOIN repositories r ON
		r.id = t.repo_id
		%s
	JOIN latest_commits_failed lcf ON
		lcf.test_id = t.id
	WHERE
		r.name = :repo
		AND r.org_id = :org_id
		AND te.created_at >= :min_created_at
		AND te.created_at <= :max_created_at
		%s
		AND te.status = "failed"
	GROUP BY t.id 
	ORDER BY failed DESC LIMIT :limit `

const latestTestExecutionQuery = `
WITH ranked_executions AS (
    SELECT
        te.test_id,
        te.duration,
        te.status,
        ROW_NUMBER() OVER(PARTITION BY te.test_id ORDER BY te.updated_at DESC) rn
    FROM
        test_execution te
    JOIN test t ON
        t.id = te.test_id
    WHERE
        t.repo_id = :repo_id
    ),
	latest_executions AS (
		SELECT
			*
		FROM
			ranked_executions
		WHERE rn = 1
	)
    SELECT
        t.id,
        t.test_locator,
		IFNULL(le.status, 'skipped') status,
		IFNULL(le.duration, 1) duration
    FROM 
        test t
    LEFT JOIN latest_executions le
        ON le.test_id = t.id
    WHERE t.repo_id = :repo_id
	%s  
	ORDER BY duration DESC
`

const findImpactedTestByBuildQuery = `
SELECT
	t.id ,
	t.test_locator,
	t.submodule
FROM
	test_execution te
JOIN test t ON
	t.id = te.test_id
WHERE
	te.build_id = :build_id
`
