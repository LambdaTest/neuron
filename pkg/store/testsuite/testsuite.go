package testsuite

import (
	"context"
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
	"gopkg.in/guregu/null.v4/zero"
)

type testSuiteStore struct {
	db     core.DB
	logger lumber.Logger
}

const insertQueryChunkSize = 1000

// New returns a new TestSuiteStore.
func New(db core.DB, logger lumber.Logger) core.TestSuiteStore {
	return &testSuiteStore{db: db, logger: logger}
}

func (t *testSuiteStore) CreateInTx(ctx context.Context, tx *sqlx.Tx, testSuiteData []*core.TestSuite) error {
	return utils.Chunk(insertQueryChunkSize, len(testSuiteData), func(start int, end int) error {
		args := []interface{}{}
		placeholderGrps := []string{}
		for _, ts := range testSuiteData[start:end] {
			placeholderGrps = append(placeholderGrps, "(?,?,?,?,?,?,?,?,?)")
			args = append(args, ts.ID, ts.Name, ts.RepoID, ts.DebutCommit, ts.ParentSuiteID, ts.Created, ts.Updated, ts.TotalTests, ts.SubModule)
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

func (t *testSuiteStore) FindByRepo(ctx context.Context, repoName, orgID, suiteID, branchName, statusText, searchText string,
	authorsNames []string, offset, limit int) ([]*core.TestSuite, error) {
	testsuites := make([]*core.TestSuite, 0)
	return testsuites, t.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo":        repoName,
			"org_id":      orgID,
			"branch":      branchName,
			"status":      statusText,
			"search_text": "%" + searchText + "%",
			"suite_id":    suiteID,
			"offset":      offset,
			"limit":       limit}

		query := listSuitesByRepoQuery
		if branchName != "" {
			query = listSuitesBranchQuery
		}
		if statusText != "" {
			query += " AND le.status = :status "
		}
		if searchText != "" {
			query += " AND ts.name LIKE :search_text"
		}
		if suiteID != "" {
			query += " AND ts.id = :suite_id "
		}
		if len(authorsNames) > 0 {
			authorWhere := ` AND gc.author_name IN (%s) `
			inClause := ""
			inClause, args = utils.AddMultipleFilters("authorNames", authorsNames, args)
			authorWhere = fmt.Sprintf(authorWhere, inClause)
			query += authorWhere
		}
		query += " ORDER BY le.end_time DESC LIMIT :limit OFFSET :offset"
		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}
		defer rows.Close()
		for rows.Next() {
			ts := new(core.TestSuite)
			ts.Meta = new(core.SuiteExecutionMeta)
			ts.Execution = new(core.TestSuiteExecution)
			ts.LatestBuild = new(core.Build)
			var totalSuiteExecution zero.Int
			if err = rows.Scan(&ts.ID,
				&ts.Name,
				&ts.ParentSuiteID,
				&ts.DebutCommit,
				&ts.TotalTests,
				&totalSuiteExecution,
				&ts.Meta.AvgTestSuiteDuration,
				&ts.Meta.Passed,
				&ts.Meta.Failed,
				&ts.Meta.Skipped,
				&ts.Meta.Blocklisted,
				&ts.Execution.SuiteID,
				&ts.Execution.CommitID,
				&ts.Execution.Status,
				&ts.Execution.Created,
				&ts.Execution.StartTime,
				&ts.Execution.EndTime,
				&ts.Execution.Duration,
				&ts.LatestBuild.CommitAuthor,
				&ts.LatestBuild.CommitMessage,
				&ts.LatestBuild.ID,
				&ts.LatestBuild.BuildNum,
				&ts.LatestBuild.Tag,
				&ts.LatestBuild.Branch); err != nil {
				return errs.SQLError(err)
			}
			if totalSuiteExecution.Valid {
				ts.Meta.Total = int(totalSuiteExecution.ValueOrZero())
			}
			testsuites = append(testsuites, ts)
		}
		if len(testsuites) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
}

func (t *testSuiteStore) FindExecution(ctx context.Context,
	testSuiteID,
	repoName,
	orgID,
	branchName, statusFilter, searchText string,
	startDate, endDate time.Time,
	offset, limit int) ([]*core.TestSuiteExecution, error) {
	testsuites := make([]*core.TestSuiteExecution, 0)
	return testsuites, t.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"suite_id":       testSuiteID,
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
		query := findTestSuiteExecutionQuery
		branchWhere := ""
		statusWhere := ""
		searchWhere := ""
		if branchName != "" {
			branchWhere = branchCondition
		}
		if statusFilter != "" {
			statusWhere = `AND tse.status = :status`
		}
		if searchText != "" {
			searchWhere = `AND ( tse.commit_id LIKE :search_text OR tse.build_id LIKE :search_text )`
		}
		query = fmt.Sprintf(query, branchWhere, statusWhere, searchWhere)
		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}
		defer rows.Close()
		for rows.Next() {
			tse := new(core.TestSuiteExecution)
			if err = rows.Scan(&tse.ID,
				&tse.BuildID,
				&tse.CommitID,
				&tse.Duration,
				&tse.Created,
				&tse.StartTime,
				&tse.EndTime,
				&tse.Status,
				&tse.BuildNum,
				&tse.BuildTag,
				&tse.Branch,
				&tse.TaskID); err != nil {
				return errs.SQLError(err)
			}
			testsuites = append(testsuites, tse)
		}

		if len(testsuites) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
}

func (t *testSuiteStore) FindStatus(ctx context.Context,
	testSuiteID,
	repoName,
	orgID,
	branchName string, startDate, endDate time.Time) ([]*core.TestSuiteExecution, error) {
	testSuites := make([]*core.TestSuiteExecution, 0)
	return testSuites, t.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"suite_id":       testSuiteID,
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
			branchJoin = `JOIN build b ON tse.build_id = b.id`
			branchWhere = branchCondition
		}
		query = fmt.Sprintf(query, branchJoin, branchWhere)
		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}
		defer rows.Close()
		for rows.Next() {
			tse := new(core.TestSuiteExecution)
			if err = rows.Scan(&tse.Created, &tse.StartTime, &tse.EndTime, &tse.Duration, &tse.Status); err != nil {
				return errs.SQLError(err)
			}
			testSuites = append(testSuites, tse)
		}
		return nil
	})
}

func (t *testSuiteStore) FindMeta(ctx context.Context, repoName, orgID, buildID, commitID,
	branchName string) ([]*core.SuiteExecutionMeta, error) {
	suites := make([]*core.SuiteExecutionMeta, 0)
	return suites, t.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo":      repoName,
			"org_id":    orgID,
			"branch":    branchName,
			"commit_id": commitID,
			"build_id":  buildID,
		}
		query := findSuiteMetaQuery
		branchBuildWhere, buildBranchJoin, commitWhere, buildWhere := "", "", "", ""
		if branchName != "" {
			branchBuildWhere = `AND b.branch_name = :branch`
			buildBranchJoin = `JOIN build b ON b.id = tse.build_id`
		}
		if commitID != "" {
			commitWhere = ` AND tse.commit_id = :commit_id`
		}
		if buildID != "" {
			buildWhere = ` AND tse.build_id = :build_id`
		}
		query = fmt.Sprintf(query, buildBranchJoin, branchBuildWhere, commitWhere, buildWhere)
		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}
		defer rows.Close()
		for rows.Next() {
			ts := new(core.SuiteExecutionMeta)
			if err = rows.Scan(&ts.Total,
				&ts.Passed,
				&ts.Failed,
				&ts.Skipped,
				&ts.Blocklisted,
				&ts.Aborted); err != nil {
				return errs.SQLError(err)
			}
			suites = append(suites, ts)
		}
		if len(suites) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
}

const branchCondition = `AND b.branch_name = :branch`

const insertQuery = `
INSERT
	INTO
	test_suite(
		id,
		name,
		repo_id,
		debut_commit,
		parent_suite_id,
		created_at,
		updated_at,
		total_tests,
		submodule
	)
VALUES %s ON
DUPLICATE KEY
UPDATE
	updated_at =
VALUES(updated_at),
total_tests =
VALUES(total_tests),
submodule = 
VALUES(submodule)
`

const listSuitesByRepoQuery = `
WITH ranked_executions AS (
	SELECT
		tse.suite_id,
		tse.commit_id,
		tse.status,
		tse.created_at, 
		tse.updated_at, 
		tse.start_time, 
		tse.end_time,
		COALESCE(tse.duration, 0) duration,
		gc.author_name,
		gc.message,
		tse.build_id,
		ROW_NUMBER() OVER (PARTITION BY tse.suite_id ORDER BY tse.updated_at DESC) rn
	FROM
		test_suite_execution tse
	JOIN test_suite ts ON
		ts.id = tse.suite_id
	JOIN repositories r ON
		r.id = ts.repo_id
	JOIN git_commits gc ON
		gc.commit_id = tse.commit_id 
		AND gc.repo_id = r.id
	WHERE
		r.name = :repo
		AND r.org_id = :org_id
),
latest_executions AS (
	SELECT
		*
	FROM
		ranked_executions
	WHERE rn = 1
),
suites_meta AS (
	SELECT
		tse2.suite_id,
		COUNT(tse2.id) counts,
		AVG(IFNULL(tse2.duration,0)) avg_test_suite_duration,
		COUNT(CASE WHEN tse2.status = 'passed' THEN tse2.id END) passed,
		COUNT(CASE WHEN tse2.status = 'failed' THEN tse2.id END) failed,
		COUNT(CASE WHEN tse2.status = 'skipped' THEN tse2.id END) skipped,
		COUNT(CASE WHEN tse2.status = 'blocklisted' THEN tse2.id END) blocklisted
	FROM
		test_suite_execution tse2 
	JOIN test_suite ts ON
		ts.id = tse2.suite_id 
	JOIN repositories r2 ON
		r2.id = ts.repo_id 
	WHERE r2.name = :repo
		AND r2.org_id = :org_id
	GROUP BY tse2.suite_id 
)
SELECT
	ts.id,
	ts.name,
	ts.parent_suite_id,
	ts.debut_commit,
	ts.total_tests,
	sm.counts,
	sm.avg_test_suite_duration,
	sm.passed,
	sm.failed,
	sm.skipped,
	sm.blocklisted,
	le.suite_id,
	le.commit_id,
	le.status,
	le.created_at,
	le.start_time,
	le.end_time,
	le.duration,
	gc.author_name,
	le.message,
	le.build_id,
	b.build_num,
	b.tag,
	b.branch_name
FROM
	test_suite ts
JOIN repositories r ON
	r.id = ts.repo_id
JOIN git_commits gc ON
	gc.commit_id = ts.debut_commit
	AND gc.repo_id = ts.repo_id
LEFT JOIN latest_executions le ON
	le.suite_id = ts.id 
JOIN build b ON
	b.id = le.build_id
JOIN suites_meta sm ON
	sm.suite_id = ts.id 
WHERE
	r.name = :repo
	AND r.org_id = :org_id
`

const listSuitesBranchQuery = `
WITH discovered_suites AS (
	SELECT
			tt.suite_id,
			tt.total_tests,
			cd.repo_id,
			cd.created_at,
			ROW_NUMBER() OVER (PARTITION BY tt.suite_id
	ORDER BY
			cd.created_at DESC) rn
	FROM
				commit_discovery cd
	JOIN JSON_TABLE(
					cd.suite_ids,
					'$[*]' COLUMNS(
						suite_id varchar(32) PATH '$.suite_id' ERROR ON
			ERROR,
						total_tests int PATH '$.total_tests' DEFAULT '0' ON
			ERROR
					)
				) tt
	JOIN repositories r ON
			r.id = cd.repo_id
	JOIN branch_commit bc ON
			bc.commit_id = cd.commit_id 
			AND bc.repo_id = r.id
	WHERE
			r.name = :repo
		AND r.org_id = :org_id
		AND bc.branch_name = :branch
		),
	suite_ids AS (
	SELECT
		DISTINCT suite_id,
		total_tests
	FROM
		discovered_suites
	WHERE
		rn = 1
	),
	ranked_executions AS (
	SELECT
			tse.suite_id,
			tse.commit_id,
			tse.status,
			tse.created_at, 
			tse.updated_at, 
			tse.start_time, 
			tse.end_time,
			COALESCE(tse.duration, 0) duration,
			gc.author_name,
			gc.message,
			tse.build_id,
			ROW_NUMBER() OVER (PARTITION BY tse.suite_id
	ORDER BY
		tse.updated_at DESC) rn
	FROM
			test_suite_execution tse
	JOIN test_suite ts ON
			ts.id = tse.suite_id
	JOIN repositories r ON
			r.id = ts.repo_id
	JOIN git_commits gc ON
			gc.commit_id = tse.commit_id
		AND gc.repo_id = r.id
	JOIN build b ON
			b.id = tse.build_id
	WHERE
			r.name = :repo
		AND r.org_id = :org_id
		AND b.branch_name = :branch
	),
	latest_executions AS (
	SELECT
			*
	FROM
			ranked_executions
	WHERE
		rn = 1
	),
	test_suite_execution_aggregates AS (
	SELECT
			tse2.suite_id,
			COUNT(tse2.id) counts,
			AVG(IFNULL(tse2.duration, 0)) avg_test_suite_duration,
			COUNT(CASE WHEN tse2.status = 'passed' THEN tse2.id END) passed,
			COUNT(CASE WHEN tse2.status = 'failed' THEN tse2.id END) failed,
			COUNT(CASE WHEN tse2.status = 'skipped' THEN tse2.id END) skipped,
			COUNT(CASE WHEN tse2.status = 'blocklisted' THEN tse2.id END) blocklisted
	FROM
			test_suite_execution tse2
	JOIN build b2 ON
			b2.id = tse2.build_id
	JOIN test_suite ts ON
			ts.id = tse2.suite_id
	JOIN repositories r2 ON
			r2.id = ts.repo_id
	WHERE
		r2.name = :repo
		AND r2.org_id = :org_id
		AND b2.branch_name = :branch
	GROUP BY
		tse2.suite_id 
	)
	SELECT
		ts.id,
		ts.name,
		ts.parent_suite_id,
		ts.debut_commit,
		si.total_tests,
		tsea.counts,
		tsea.avg_test_suite_duration,
		tsea.passed,
		tsea.failed,
		tsea.skipped,
		tsea.blocklisted,
		le.suite_id,
		le.commit_id,
		le.status,
		le.created_at,
		le.start_time,
		le.end_time,
		le.duration,
		gc.author_name,
		le.message,
		le.build_id,
		b.build_num,
		b.tag,
		b.branch_name
	FROM
		test_suite ts
	JOIN test_suite_branch tsb ON
		tsb.test_suite_id = ts.id
	JOIN repositories r ON
		r.id = ts.repo_id
	JOIN git_commits gc ON
		gc.commit_id = ts.debut_commit
		AND gc.repo_id = ts.repo_id
	LEFT JOIN latest_executions le ON
		le.suite_id = ts.id
	JOIN build b ON
		b.id = le.build_id
	JOIN test_suite_execution_aggregates tsea ON
		tsea.suite_id = ts.id
	JOIN suite_ids si ON
		si.suite_id = ts.id
	WHERE
		r.name = :repo
		AND r.org_id = :org_id
		AND tsb.branch_name = :branch
`

const findStatusQuery = `
SELECT
	tse.created_at,
	tse.start_time,
	tse.end_time,
	tse.duration,
	tse.status
FROM
	test_suite ts
JOIN test_suite_execution tse ON 
	tse.suite_id = ts.id
JOIN repositories r ON
	r.id = ts.repo_id
%s
WHERE
	ts.id = :suite_id
	AND r.name = :repo
	AND r.org_id = :org_id
	AND tse.start_time >= :min_created_at
	AND tse.start_time <= :max_created_at
	%s
`

const findTestSuiteExecutionQuery = `
SELECT
	tse.id,
	tse.build_id,
	tse.commit_id,
	tse.duration,
	tse.created_at,
	tse.start_time,
	tse.end_time,
	tse.status,
	b.build_num,
	b.tag,
	b.branch_name,
	tse.task_id
FROM
	test_suite ts
JOIN test_suite_execution tse ON
	tse.suite_id = ts.id
JOIN repositories r ON
	r.id = ts.repo_id
JOIN build b ON
	tse.build_id = b.id
WHERE
	ts.id = :suite_id
	AND r.name = :repo
	AND r.org_id = :org_id
	AND tse.created_at >= :min_created_at
	AND tse.created_at <= :max_created_at
	%s
	%s
	%s
ORDER BY tse.updated_at DESC
LIMIT :limit OFFSET :offset
`

const findSuiteMetaQuery = `
WITH ranked_executions AS (
	SELECT
		tse.suite_id,
		tse.status,
		ROW_NUMBER() OVER (PARTITION BY tse.suite_id ORDER BY tse.updated_at DESC) rn
	FROM
		test_suite_execution tse
	JOIN test_suite ts ON
		ts.id = tse.suite_id
	JOIN repositories r ON
		r.id = ts.repo_id
	%s
	WHERE
		r.name = :repo
		AND r.org_id = :org_id
		%s
		%s
		%s
)
SELECT
	COALESCE(COUNT(DISTINCT tse.suite_id),0) total,
	COALESCE(COUNT(DISTINCT CASE WHEN tse.status = 'passed' THEN tse.suite_id END),0) passed,
	COALESCE(COUNT(DISTINCT CASE WHEN tse.status = 'failed' THEN tse.suite_id END),0) failed,
	COALESCE(COUNT(DISTINCT CASE WHEN tse.status = 'skipped' THEN tse.suite_id END),0) skipped,
	COALESCE(COUNT(DISTINCT CASE WHEN tse.status = 'blocklisted' THEN tse.suite_id END),0) blocklisted,
	COALESCE(COUNT(DISTINCT CASE WHEN tse.status = 'aborted' THEN tse.suite_id END),0) aborted
	FROM
		ranked_executions tse
	WHERE
		rn = 1
`
