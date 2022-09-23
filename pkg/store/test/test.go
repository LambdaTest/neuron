package test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"sort"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/gocraft/dbr"
	"github.com/gocraft/dbr/dialect"
	"github.com/jmoiron/sqlx"
)

type testStore struct {
	db     core.DB
	logger lumber.Logger
}

const insertQueryChunkSize = 1000

// New returns a new TestStore.
func New(db core.DB, logger lumber.Logger) core.TestStore {
	return &testStore{db: db, logger: logger}
}

func (ts *testStore) CreateInTx(ctx context.Context, tx *sqlx.Tx, testData []*core.Test) error {
	return utils.Chunk(insertQueryChunkSize, len(testData), func(start int, end int) error {
		args := []interface{}{}
		placeholderGrps := []string{}
		for _, test := range testData[start:end] {
			placeholderGrps = append(placeholderGrps, "(?,?,?,?,?,?,?,?,?)")
			args = append(args, test.ID, test.Name, test.TestSuiteID, test.RepoID, test.DebutCommit,
				test.TestLocator, test.Created, test.Updated, test.SubModule)
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

func (ts *testStore) FindByRepo(ctx context.Context, repoName, orgID, testID, branchName, statusFilter, searchText string,
	authorsNames []string, offset, limit int) ([]*core.Test, error) {
	tests := make([]*core.Test, 0, limit)
	return tests, ts.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo":        repoName,
			"org_id":      orgID,
			"branch":      branchName,
			"status":      statusFilter,
			"search_text": "%" + searchText + "%",
			"test_id":     testID,
			"offset":      offset,
			"limit":       limit}
		query := listTestByRepoQuery
		branchBuildWhere, branchTestWhere, buildBranchJoin, buildBranchJoin2, testbranchJoin := "", "", "", "", ""
		statusWhere, searchWhere, testIDWhere, authorWhere, branchBuildJoin3 := "", "", "", "", ""
		if branchName != "" {
			branchBuildWhere = `AND b.branch_name = :branch`
			branchTestWhere = `AND tb.branch_name = :branch`
			buildBranchJoin = `JOIN build b ON b.id = te.build_id`
			buildBranchJoin2 = `JOIN build b ON b.id = te2.build_id`
			testbranchJoin = `JOIN test_branch tb ON tb.test_id = t.id`
			branchBuildJoin3 = `JOIN build b ON b.id = fte2.build_id`
		}
		if statusFilter != "" {
			statusWhere = `AND le.status = :status`
		}
		if searchText != "" {
			searchWhere = `AND t.name LIKE :search_text`
		}
		if testID != "" {
			testIDWhere = `AND t.id = :test_id`
		}
		if len(authorsNames) > 0 {
			authorWhere = ` AND gc.author_name IN (%s) `
			inClause := ""
			inClause, args = utils.AddMultipleFilters("auhtorNames", authorsNames, args)
			authorWhere = fmt.Sprintf(authorWhere, inClause)
		}
		query = fmt.Sprintf(query, buildBranchJoin2, branchBuildWhere,
			buildBranchJoin, branchBuildWhere, branchBuildWhere,
			branchBuildWhere, branchBuildJoin3, branchBuildWhere, testbranchJoin, branchTestWhere,
			statusWhere, searchWhere, testIDWhere, authorWhere)
		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}
		defer rows.Close()
		for rows.Next() {
			t := new(core.Test)
			t.Meta = new(core.ExecutionMeta)
			t.FlakyMeta = new(core.FlakyExecutionMetadata)
			t.Execution = new(core.TestExecution)
			t.Transition = new(core.Transition)
			var jsonRaw string
			var statuses []*core.TestStatus
			if err = rows.Scan(&t.ID, &t.Name, &t.TestSuiteID, &t.TestSuiteName, &t.TestLocator, &t.DebutCommit, &t.Meta.Total,
				&t.Meta.Passed, &t.Meta.Failed, &t.Meta.Skipped, &t.Meta.Blocklisted, &t.Meta.Quarantined, &t.Meta.Aborted, &t.Meta.AvgTestDuration,
				&t.FlakyMeta.ImpactedTests, &t.FlakyMeta.FlakyTests, &t.FlakyMeta.Skipped, &t.FlakyMeta.Blocklisted, &t.FlakyMeta.Aborted,
				&t.Execution.CommitAuthor, &t.Execution.CommitMessage, &jsonRaw, &t.Execution.CommitID,
				&t.Execution.Status, &t.Execution.Created, &t.Execution.Updated, &t.Execution.StartTime, &t.Execution.EndTime,
				&t.Execution.Duration, &t.Execution.BlocklistSource, &t.Execution.BuildID,
				&t.Execution.TaskID, &t.Execution.BuildNum, &t.Execution.BuildTag,
				&t.Execution.Branch, &t.Created, &t.FlakyMeta.LastFlakeStatus, &t.FlakyMeta.LastFlakeTime,
			); err != nil {
				return errs.SQLError(err)
			}
			if t.FlakyMeta.ImpactedTests == 0 {
				t.FlakyMeta.OverAllFlakiness = 0
			} else {
				t.FlakyMeta.OverAllFlakiness = float32(t.FlakyMeta.FlakyTests) / float32(t.FlakyMeta.ImpactedTests)
			}
			if err := json.Unmarshal([]byte(jsonRaw), &statuses); err != nil {
				ts.logger.Errorf("failed to unmarshall json for testID %s, repoName %s,orgID %s, error %v", t.ID, repoName, orgID, err)
				return err
			}
			if len(statuses) > 1 {
				sort.Slice(statuses, func(i, j int) bool {
					return statuses[i].Created > statuses[j].Created
				})
				t.Transition.CurrentStatus = statuses[0].Status.String
				t.Transition.PreviousStatus = statuses[1].Status.String
				if t.Execution.Status != core.TestExecutionStatus(t.Transition.CurrentStatus) && statuses[0].Created == statuses[1].Created {
					t.Transition.PreviousStatus = t.Transition.CurrentStatus
					t.Transition.CurrentStatus = string(t.Execution.Status)
				}
			} else if len(statuses) == 1 {
				t.Transition.CurrentStatus = statuses[0].Status.String
				t.Transition.PreviousStatus = ""
			} else {
				t.Transition.CurrentStatus = ""
				t.Transition.PreviousStatus = ""
			}
			tests = append(tests, t)
		}
		if len(tests) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
}

//TODO: optimize
func (ts *testStore) FindRepoTestStatuses(ctx context.Context, repoName, orgID, branchName, status, tag, searchText,
	lastSeenID string, startDate, endDate time.Time, limit int, isJobs bool) ([]*core.Test, error) {
	tests := make([]*core.Test, 0, limit)
	return tests, ts.db.Execute(func(db *sqlx.DB) error {
		query, groupByClause, args := repoTestStatusQueryFilters(branchName, status, tag, repoName, orgID, isJobs, startDate, endDate)
		if searchText != "" {
			query += " AND t.name LIKE ? "
			args = append(args, "%"+searchText+"%")
		}
		if lastSeenID != "" {
			query += testLastSeen
			args = append(args, lastSeenID)
		}
		args = append(args, limit)
		query += groupByClause

		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			return errs.SQLError(err)
		}
		if rows.Err() != nil {
			return rows.Err()
		}
		defer rows.Close()
		for rows.Next() {
			t := new(core.Test)
			var jsonRaw string
			var statuses []*core.TestStatus
			if err = rows.Scan(&t.ID, &t.Name, &t.TestSuiteID, &t.TestLocator, &jsonRaw, &t.Introduced); err != nil {
				return errs.SQLError(err)
			}
			if err := json.Unmarshal([]byte(jsonRaw), &statuses); err != nil {
				ts.logger.Errorf("failed to unmarshall json for testID %s, repoName %s,orgID %s, error %v", t.ID, repoName, orgID, err)
				return err
			}
			if len(statuses) > 1 {
				sort.Slice(statuses, func(i, j int) bool {
					return statuses[i].Created > statuses[j].Created
				})
			}
			t.TestStatus = statuses
			tests = append(tests, t)
		}

		if len(tests) == 0 {
			return errs.ErrRowsNotFound
		}

		return nil
	})
}

func repoTestStatusQueryFilters(branchName, status, tag, repoName, orgID string, isJobs bool, startDate,
	endDate time.Time) (query, groupByClause string, args []interface{}) {
	args = []interface{}{startDate, endDate, repoName, orgID}
	if isJobs {
		query = repoTestJobsQuery
		if branchName != "" {
			query += " AND b.branch_name = ? "
			args = append(args, branchName)
		}
		if status != "" {
			query += " AND b.status = ? "
			args = append(args, status)
		}
		if tag != "" {
			query += " AND b.tag = ? "
			args = append(args, tag)
		}
		groupByClause = " GROUP BY t.id ORDER BY MAX(b.created_at) DESC, t.id DESC LIMIT ? "
	} else {
		query = repoTestStatusQuery
		if branchName != "" {
			query = repoTestStatusBranchQuery
			args = append(args, branchName)
		}
		groupByClause = " GROUP BY t.id ORDER BY MAX(te.updated_at) DESC, t.id DESC LIMIT ? "
	}
	return query, groupByClause, args
}

func (ts *testStore) FindUnimpactedTests(ctx context.Context,
	commitID,
	buildID,
	repoName,
	orgID string, offset, limit int) ([]*core.Test, error) {
	tests := make([]*core.Test, 0, limit)
	return tests, ts.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo":      repoName,
			"org_id":    orgID,
			"commit_id": commitID,
			"build_id":  buildID,
			"limit":     limit,
			"offset":    offset,
		}
		query := unimpactedTestsQuery
		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}
		if rows.Err() != nil {
			return rows.Err()
		}
		defer rows.Close()
		for rows.Next() {
			t := new(core.Test)
			if err = rows.Scan(&t.ID, &t.Name, &t.TestSuiteID, &t.DebutCommit, &t.RepoID, &t.TestLocator, &t.Created, &t.Updated, &t.TestSuiteName); err != nil {
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

func (ts *testStore) FindIntroducedDate(ctx context.Context, repoName, orgID, branchName,
	lastSeenID string, limit int) (map[string]time.Time, error) {
	introducedMap := make(map[string]time.Time)
	return introducedMap, ts.db.Execute(func(db *sqlx.DB) error {
		args := []interface{}{repoName, orgID}
		query := introducedDateListQuery
		if branchName != "" {
			query = introducedDateListBranchQuery
			args = append(args, branchName)
		}
		if lastSeenID != "" {
			query += " AND t.id <= ? "
			args = append(args, lastSeenID)
		}
		query += " ORDER BY t.id DESC LIMIT ?"
		args = append(args, limit)

		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			return errs.SQLError(err)
		}
		if rows.Err() != nil {
			return rows.Err()
		}

		defer rows.Close()
		for rows.Next() {
			var testID string
			var createdAtTime time.Time
			if err = rows.Scan(&testID, &createdAtTime); err != nil {
				return errs.SQLError(err)
			}
			_, exists := introducedMap[testID]
			if !exists {
				introducedMap[testID] = createdAtTime
			}
		}
		return nil
	})
}

func (ts *testStore) FindRepoTestStatusesSlowest(ctx context.Context, repoName, orgID, branchName string,
	startDate, endDate time.Time, limit int) ([]*core.Test, error) {
	tests := make([]*core.Test, 0, limit)
	return tests, ts.db.Execute(func(db *sqlx.DB) error {
		args := []interface{}{repoName, orgID, startDate, endDate}
		query := findRepoTestStatusSlowestQuery
		if branchName != "" {
			query = fmt.Sprintf(query, " JOIN build b ON b.id = te.build_id ", " AND b.branch_name = ? ")
			args = append(args, branchName)
		} else {
			query = fmt.Sprintf(query, "", "")
		}
		args = append(args, limit)
		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			return errs.SQLError(err)
		}
		if rows.Err() != nil {
			return rows.Err()
		}
		defer rows.Close()
		for rows.Next() {
			t := new(core.Test)
			t.Execution = new(core.TestExecution)
			if err = rows.Scan(&t.ID,
				&t.DebutCommit,
				&t.Name,
				&t.Execution.Duration,
				&t.Execution.Created,
				&t.Execution.CommitAuthor); err != nil {
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

func (ts *testStore) FindTestData(ctx context.Context,
	repoName,
	orgID,
	branchName string,
	startDate, endDate time.Time) ([]*core.GitCommit, error) {
	testData := make([]*core.GitCommit, 0)
	return testData, ts.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo":       repoName,
			"org_id":     orgID,
			"start_date": startDate,
			"end_date":   endDate,
			"branch":     branchName,
		}
		query := findTestDataQuery
		if branchName != "" {
			query = fmt.Sprintf(query, ` JOIN branch_commit bc ON
			bc.commit_id = gc3.commit_id 
			AND bc.repo_id = r3.id `, ` AND bc.branch_name = :branch `)
		} else {
			query = fmt.Sprintf(query, "", "")
		}
		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}
		if rows.Err() != nil {
			return rows.Err()
		}
		defer rows.Close()
		for rows.Next() {
			gc := new(core.GitCommit)
			gc.TestMeta = new(core.ExecutionMeta)
			gc.FlakyMeta = new(core.FlakyExecutionMetadata)
			if err := rows.Scan(&gc.TestMeta.Total,
				&gc.TestMeta.Blocklisted,
				&gc.TestMeta.Quarantined,
				&gc.TestMeta.TotalTests,
				&gc.FlakyMeta.ImpactedTests,
				&gc.FlakyMeta.FlakyTests,
				&gc.FlakyMeta.Skipped,
				&gc.FlakyMeta.Blocklisted,
				&gc.FlakyMeta.Aborted,
				&gc.CommitID,
				&gc.Created); err != nil {
				return errs.SQLError(err)
			}
			if gc.FlakyMeta.ImpactedTests == 0 {
				gc.FlakyMeta.OverAllFlakiness = 0
			} else {
				gc.FlakyMeta.OverAllFlakiness = float32(gc.FlakyMeta.FlakyTests) / float32(gc.FlakyMeta.ImpactedTests)
			}
			testData = append(testData, gc)
		}

		if len(testData) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
}

func (ts *testStore) FindTestMeta(ctx context.Context, repoName, orgID, buildID, commitID,
	branchName string) ([]*core.ExecutionMeta, error) {
	tests := make([]*core.ExecutionMeta, 0)
	return tests, ts.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo":      repoName,
			"org_id":    orgID,
			"branch":    branchName,
			"commit_id": commitID,
			"build_id":  buildID,
		}
		query := findTestMetaQuery
		branchBuildWhere, buildBranchJoin, commitWhere, buildWhere, buildBranchFlakyJoin := "", "", "", "", ""
		buildWhereFlaky, commitWhereFlaky := "", ""

		if branchName != "" {
			branchBuildWhere = `AND b.branch_name = :branch`
			buildBranchJoin = `JOIN build b ON b.id = te.build_id`
			buildBranchFlakyJoin = `JOIN build b ON b.id = fte.build_id`
		}
		if commitID != "" {
			commitWhere = ` AND te.commit_id = :commit_id`
			commitWhereFlaky = ` AND b.commit_id = :commit_id`
		}
		if buildID != "" {
			buildWhere = ` AND te.build_id = :build_id`
			buildWhereFlaky = ` AND fte.build_id = :build_id`
		}
		query = fmt.Sprintf(query, buildBranchJoin, branchBuildWhere, commitWhere, buildWhere, buildBranchFlakyJoin,
			branchBuildWhere, commitWhereFlaky, buildWhereFlaky)
		fmt.Println(query)
		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}
		defer rows.Close()
		for rows.Next() {
			t := new(core.ExecutionMeta)
			if err = rows.Scan(&t.Total,
				&t.Passed,
				&t.Failed,
				&t.Skipped,
				&t.Blocklisted,
				&t.Quarantined,
				&t.Aborted,
				&t.Flaky,
				&t.NonFlaky); err != nil {
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

// FindBlocklistedTests returns the number of blocklisted tests by an author
func (ts *testStore) FindBlocklistedTests(ctx context.Context,
	repoName, orgID, branchName, authorName string,
	startDate, endDate time.Time) (blocklistedTests int, err error) {
	blocklistedTests = 0
	errorExec := ts.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo":       repoName,
			"org":        orgID,
			"author":     authorName,
			"branch":     branchName,
			"start_date": startDate,
			"end_date":   endDate,
		}
		branchCommitJoin, branchCommitWhere := "", ""
		if branchName != "" {
			branchCommitJoin = ` JOIN branch_commit bc ON
			bc.commit_id = gc.commit_id `
			branchCommitWhere = " AND bc.branch_name = :branch "
		}
		query := findBlocklistedTestsQuery
		query = fmt.Sprintf(query, branchCommitJoin, branchCommitWhere)
		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}
		defer rows.Close()
		for rows.Next() {
			if err = rows.Scan(&blocklistedTests); err != nil {
				return errs.SQLError(err)
			}
		}
		return nil
	})
	return blocklistedTests, errorExec
}

func (ts *testStore) FindIrregularTests(ctx context.Context, repoID, branchName string,
	startDate, endDate time.Time, limit int) ([]*core.IrregularTests, error) {
	irrTests := make([]*core.IrregularTests, 0, limit)
	err := ts.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo_id":        repoID,
			"branch":         branchName,
			"min_created_at": startDate,
			"max_created_at": endDate,
			"limit":          limit,
		}
		query := findRepoIrregularTests
		branchBuildWhere := ""
		if branchName != "" {
			branchBuildWhere = " AND b.branch_name = :branch"
		}
		query = fmt.Sprintf(query, branchBuildWhere)
		rows, errS := db.NamedQueryContext(ctx, query, args)
		if errS != nil {
			return errs.SQLError(errS)
		}
		defer rows.Close()
		for rows.Next() {
			irrT := new(core.IrregularTests)
			if errR := rows.StructScan(&irrT); errR != nil {
				return errs.SQLError(errR)
			}
			irrTests = append(irrTests, irrT)
		}
		if len(irrTests) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
	if err != nil {
		return nil, errs.SQLError(err)
	}
	return irrTests, nil
}

func (ts *testStore) FindMonthWiseNetTests(ctx context.Context, repoID, branchName string,
	startDate, endDate time.Time) ([]*core.MonthWiseNetTests, error) {
	testsData := make([]*core.MonthWiseNetTests, 0)
	err := ts.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo_id":        repoID,
			"branch":         branchName,
			"min_created_at": startDate,
			"max_created_at": endDate,
		}
		query := findTestCountMonthWise
		branchCommitJoin := ""
		branchCommitWhere := ""
		if branchName != "" {
			branchCommitJoin = "JOIN branch_commit bc ON bc.commit_id = cd.commit_id"
			branchCommitWhere = "AND bc.branch_name = :branch"
		}
		query = fmt.Sprintf(query, branchCommitJoin, branchCommitWhere)
		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}

		defer rows.Close()
		for rows.Next() {
			ts := new(core.MonthWiseNetTests)
			if err := rows.StructScan(ts); err != nil {
				return errs.SQLError(err)
			}
			testsData = append(testsData, ts)
		}
		if rows.Err() != nil {
			return rows.Err()
		}

		if len(testsData) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})

	if err != nil {
		return nil, errs.SQLError(err)
	}
	return testsData, nil
}

func (ts *testStore) FindAddedTests(ctx context.Context, repoID, branchName string,
	startDate, endDate time.Time) (int64, error) {
	var addedTestCount int64 = 0
	err := ts.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo_id":        repoID,
			"min_created_at": startDate,
			"max_created_at": endDate,
			"branch":         branchName,
		}
		query := findAddedTests
		branchCommitJoinGC := ""
		branchCommitJoinCD := ""
		branchCommitWhere := ""
		if branchName != "" {
			branchCommitJoinGC = " JOIN branch_commit bc ON bc.repo_id = gc.repo_id AND bc.commit_id = gc.commit_id"
			branchCommitJoinCD = " JOIN branch_commit bc ON bc.repo_id = cd.repo_id AND bc.commit_id = cd.commit_id"
			branchCommitWhere = " AND bc.branch_name = :branch"
		}
		query = fmt.Sprintf(query, branchCommitJoinGC, branchCommitWhere, branchCommitJoinCD,
			branchCommitWhere, branchCommitJoinCD, branchCommitWhere)
		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}
		defer rows.Close()
		for rows.Next() {
			if err = rows.Scan(&addedTestCount); err != nil {
				return errs.SQLError(err)
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return addedTestCount, nil
}

const testLastSeen = " AND t.id <= ? "

const repoTestStatusQuery = `
SELECT
	t.id,
	t.name,
	t.test_suite_id,
	t.test_locator,
	JSON_ARRAYAGG(JSON_OBJECT('commit_id',
	gc.commit_id,
	'author_name',
	gc.author_name,
	'status',
	te.status,
	'created_at',
	UNIX_TIMESTAMP(te.created_at),
	'start_time',
	UNIX_TIMESTAMP(te.start_time),
	'end_time',
	UNIX_TIMESTAMP(te.end_time))) status,
	MIN(t.created_at)
FROM
	test t
JOIN repositories r ON
	r.id = t.repo_id
JOIN git_commits gc ON
	gc.repo_id = r.id
JOIN test_execution te ON
	te.test_id = t.id
	AND te.commit_id = gc.commit_id 
JOIN build b ON
	b.id = te.build_id 
WHERE 
	b.created_at >= ?
	AND b.created_at <= ?
	AND r.name =?
	AND r.org_id =?
`

const repoTestStatusBranchQuery = `
SELECT
	t.id,
	t.name,
	t.test_suite_id,
	t.test_locator,
	JSON_ARRAYAGG(JSON_OBJECT('commit_id',
	gc.commit_id,
	'author_name',
	gc.author_name,
	'status',
	te.status,
	'created_at',
	UNIX_TIMESTAMP(te.created_at),
	'start_time',
	UNIX_TIMESTAMP(te.start_time),
	'end_time',
	UNIX_TIMESTAMP(te.end_time),
	'branch', COALESCE(bc.branch_name, ""))) status,
	MIN(t.created_at)
FROM
	test t
JOIN repositories r ON
	r.id = t.repo_id
JOIN git_commits gc ON
	gc.repo_id = r.id
JOIN test_execution te ON
	te.test_id = t.id
	AND te.commit_id = gc.commit_id 
JOIN branch_commit bc ON
	bc.commit_id = gc.commit_id 
	AND bc.repo_id = r.id 
WHERE
	gc.created_at >= ?
	AND gc.created_at <= ?
	AND r.name =?
	AND r.org_id =?
	AND bc.branch_name = ?
`

const insertQuery = `
INSERT
	INTO
	test(
		id,
		name,
		test_suite_id,
		repo_id,
		debut_commit,
		test_locator,
		created_at,
		updated_at,
		submodule
	)
VALUES %s ON
DUPLICATE KEY
UPDATE
	updated_at =
VALUES(updated_at),
	submodule = 
VALUES(submodule)
`

const listTestByRepoQuery = `
WITH ranked_transitions AS (
	SELECT
			te2.test_id ts_id,
			te2.status,
			te2.created_at,
			ROW_NUMBER() OVER (PARTITION BY te2.test_id
	ORDER BY
			te2.updated_at DESC) rn
	FROM
			test_execution te2
	JOIN test t ON
			t.id = te2.test_id
	JOIN repositories r ON
			t.repo_id = r.id
			%s
	WHERE
			r.name = :repo
		AND r.org_id = :org_id
			%s
		),
		transitions AS (
	SELECT
			rs.ts_id ts_id,
				JSON_ARRAYAGG(
					JSON_OBJECT(
						'status',
						rs.status,
						'created_at',
						UNIX_TIMESTAMP(rs.created_at)
					)
				) transitions
	FROM
			ranked_transitions rs
	WHERE
			rs.rn <= 2
	GROUP BY
			ts_id
	),
	ranked_executions AS (
	SELECT
			te.test_id,
			te.commit_id,
			te.status,
			te.created_at,
			te.updated_at,
			te.start_time,
			te.end_time,
			COALESCE(te.duration, 0) duration,
			te.blocklist_source,
			te.build_id,
			te.task_id,
			ROW_NUMBER() OVER (PARTITION BY te.test_id
	ORDER BY
		te.updated_at DESC) rn
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
			%s
	),
	latest_executions AS (
	SELECT
			*
	FROM
			ranked_executions
	WHERE
			rn = 1
	),
	test_execution_aggregates AS (
	SELECT
			t.id test_id,
			COUNT(te.id) total,
			COUNT(CASE WHEN te.status = 'passed' THEN te.id END) passed,
			COUNT(CASE WHEN te.status = 'failed' THEN te.id END) failed,
			COUNT(CASE WHEN te.status = 'skipped' THEN te.id END) skipped,
			COUNT(CASE WHEN te.status = 'blocklisted' THEN te.id END) blocklisted,
			COUNT(CASE WHEN te.status = 'quarantined' THEN te.id END) quarantined,
			COUNT(CASE WHEN te.status = 'aborted' THEN te.id END) aborted,
			AVG(IFNULL(te.duration, 0)) avg_duration
	FROM
			test_execution te
	JOIN build b ON
			b.id = te.build_id
	JOIN test t ON
			t.id = te.test_id
		AND t.repo_id = b.repo_id
	JOIN repositories r ON
			r.id = t.repo_id
	WHERE
			r.name = :repo
		AND r.org_id = :org_id
			%s
	GROUP BY
		t.id
	),
	flaky_test_execution_aggregates AS (
	SELECT
		fte.test_id,
		COUNT(DISTINCT(fte.test_id)) as impacted_tests,
		COUNT(CASE WHEN fte.status = 'flaky' THEN fte.id END) as flaky_tests,
		COUNT(CASE WHEN fte.status = 'skipped' THEN fte.id END) skipped,
		COUNT(CASE WHEN fte.status = 'blocklisted' THEN fte.id END) blocklisted,
		COUNT(CASE WHEN fte.status = 'aborted' THEN fte.id END) aborted
	FROM
		flaky_test_execution fte
	JOIN build b ON
		b.id = fte.build_id
	JOIN test t ON
		t.id = fte.test_id
		AND t.repo_id = b.repo_id
	JOIN repositories r ON
		r.id = t.repo_id
	WHERE
		r.name = :repo
		AND r.org_id = :org_id
		%s
	GROUP BY
		t.id
    ),
	flaky_ranked_executions AS (
		SELECT
				fte2.test_id,
				fte2.status,
				fte2.updated_at,
				ROW_NUMBER() OVER (PARTITION BY fte2.test_id
		ORDER BY
			fte2.updated_at DESC) rn
		FROM
				flaky_test_execution fte2
		JOIN test t ON
				t.id = fte2.test_id
		JOIN repositories r ON
				r.id = t.repo_id
				%s
		WHERE
			r.name = :repo
			AND r.org_id = :org_id
			%s
		),
		flaky_latest_executions AS (
		SELECT
				*
		FROM
				flaky_ranked_executions
		WHERE
				rn = 1
		)
	SELECT
		t.id,
		t.name,
		t.test_suite_id,
		ts.name,
		t.test_locator,
		t.debut_commit,
		COALESCE(tea.total, 0),
		COALESCE(tea.passed, 0),
		COALESCE(tea.failed, 0),
		COALESCE(tea.skipped, 0),
		COALESCE(tea.blocklisted, 0),
		COALESCE(tea.quarantined, 0),
		COALESCE(tea.aborted, 0),
		COALESCE(tea.avg_duration, 0),
		COALESCE(ftea.impacted_tests, 0) flaky_impacted_tests,
		COALESCE(ftea.flaky_tests, 0) flaky_tests,
		COALESCE(ftea.skipped, 0) flaky_skipped,
		COALESCE(ftea.blocklisted, 0) flaky_blocklisted,
		COALESCE(ftea.aborted, 0) flaky_aborted,
		gc.author_name,
		gc.message,
		COALESCE(trs.transitions, JSON_ARRAY()),
		COALESCE(le.commit_id, ""),
		COALESCE(le.status, ""),
		COALESCE(le.created_at, TIMESTAMP("0001-01-01")),
		COALESCE(le.updated_at, TIMESTAMP("0001-01-01")),
		COALESCE(le.start_time, TIMESTAMP("0001-01-01")),
		COALESCE(le.end_time, TIMESTAMP("0001-01-01")),
		COALESCE(le.duration, 0) duration,
		COALESCE(le.blocklist_source, ""),
		COALESCE(le.build_id, ""),
		COALESCE(le.task_id, ""),
		COALESCE(b.build_num, 0),
		COALESCE(b.tag, ""),
		COALESCE(b.branch_name, ""),
		t.created_at,
		COALESCE(fle.status, "") latest_flaky_status,
		COALESCE(fle.updated_at, TIMESTAMP("0001-01-01")) latest_flaky_time
	FROM
		test t
	%s
	LEFT JOIN test_suite ts ON
		ts.id = t.test_suite_id
	JOIN repositories r ON
		r.id = t.repo_id
	JOIN git_commits gc ON
		gc.commit_id = t.debut_commit
		AND gc.repo_id = t.repo_id
	LEFT JOIN transitions trs ON
		trs.ts_id = t.id
	LEFT JOIN latest_executions le ON
		le.test_id = t.id
	LEFT JOIN build b ON
		b.id = le.build_id
	LEFT JOIN test_execution_aggregates tea ON
		tea.test_id = t.id
	LEFT JOIN flaky_test_execution_aggregates ftea ON
		ftea.test_id = t.id	
	LEFT JOIN flaky_latest_executions fle ON
		fle.test_id = t.id 
	WHERE
		r.name = :repo
		AND r.org_id = :org_id
		%s
		%s
		%s
		%s
		%s
	ORDER BY
		le.updated_at DESC,
		t.id DESC
	LIMIT :limit OFFSET :offset
`

const introducedDateListQuery = `
SELECT
	t.id,
	t.created_at 
FROM
	test t
JOIN repositories r ON
	r.id = t.repo_id
WHERE
	r.name = ?
	AND r.org_id = ?
`
const introducedDateListBranchQuery = `
SELECT
	t.id,
	t.created_at 
FROM
	test t
JOIN repositories r ON
	r.id = t.repo_id
JOIN test_branch tb ON
	tb.test_id = t.id 
	AND tb.repo_id = r.id 
WHERE
	r.name = ?
	AND r.org_id = ?
	AND tb.branch_name = ?
`

const unimpactedTestsQuery = `
WITH impacted_tests AS(
SELECT
	te.test_id
FROM
	test_execution te
JOIN build b ON
	te.build_id = b.id
WHERE
	b.commit_id = :commit_id 
	AND te.build_id = :build_id
UNION 
SELECT
	fte.test_id
FROM
	flaky_test_execution fte 
JOIN build b ON
	fte.build_id = b.id
WHERE
	b.commit_id = :commit_id 
	AND fte.build_id = :build_id
)
SELECT 
	test.id,
	test.name,
	test.test_suite_id,
	test.debut_commit,
	test.repo_id,
	test.test_locator,
	test.created_at,
	test.updated_at,
	ts.name
FROM 
	test
JOIN repositories r ON
	r.id = test.repo_id
JOIN (
		SELECT
			tt.test_id
		FROM
			commit_discovery cd
		JOIN
		JSON_TABLE(
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
			cd.commit_id = :commit_id
			AND b.id = :build_id
			AND tt.test_id NOT IN (
				SELECT 
					test_id
				FROM
					impacted_tests
			)
	) unimpacted_test_ids ON
	unimpacted_test_ids.test_id = test.id
LEFT JOIN test_suite ts ON
	test.test_suite_id = ts.id
WHERE
	r.name = :repo
	AND r.org_id = :org_id
ORDER BY
	test.created_at DESC
LIMIT :limit OFFSET :offset
`

const findRepoTestStatusSlowestQuery = `
WITH ranked_executions AS(
	SELECT
		te.test_id,
		te.commit_id,
		t.name,
		te.duration duration,
		te.updated_at latest_execution,
		gc.author_name,
		ROW_NUMBER() OVER (PARTITION BY te.test_id
	ORDER BY
		te.duration DESC) rn
	FROM
		test_execution te
	JOIN test t ON
		t.id = te.test_id
	JOIN repositories r ON
		r.id = t.repo_id
	JOIN git_commits gc ON
		gc.commit_id = t.debut_commit
		AND gc.repo_id = r.id
		%s
	WHERE
		r.name = ?
		AND r.org_id = ?
		AND te.created_at >= ?
		AND te.created_at <= ?
		%s
	)
	SELECT
		test_id,
		commit_id,
		name,
		duration,
		latest_execution,
		author_name
	FROM
		ranked_executions
	WHERE
		rn = 1
	ORDER BY
		duration DESC
	LIMIT ?
`

const findTestDataQuery = `
WITH discovery AS (
	SELECT
	cd.commit_id,
	tt.test_id
FROM
	commit_discovery cd
JOIN JSON_TABLE(
				cd.test_ids,
				'$[*]' COLUMNS(
					test_id varchar(32) PATH '$' ERROR ON
		ERROR
		)
			) tt
JOIN git_commits gc ON
	gc.commit_id = cd.commit_id
	AND gc.repo_id = cd.repo_id
JOIN repositories r ON
	r.id = gc.repo_id
WHERE
	r.name = :repo
	AND r.org_id = :org_id
	AND gc.created_at >= :start_date
	AND gc.created_at <= :end_date 
),
execution AS (
	SELECT
	COUNT(DISTINCT te.test_id) impacted_tests,
	COUNT(DISTINCT CASE WHEN te.status = 'blocklisted' THEN te.id END) blocklisted,
	COUNT(DISTINCT CASE WHEN te.status = 'quarantined' THEN te.id END) quarantined,
	gc2.commit_id
FROM
	test_execution te
JOIN git_commits gc2 
	ON
	gc2.commit_id = te.commit_id
JOIN repositories r2 ON
	gc2.repo_id = r2.id
JOIN discovery d ON
	d.commit_id = gc2.commit_id 
	AND d.test_id = te.test_id 
WHERE
	r2.name = :repo
	AND r2.org_id = :org_id
	AND gc2.created_at >= :start_date
	AND gc2.created_at <= :end_date
	GROUP BY gc2.commit_id 
),
flaky_execution AS (
	SELECT
	COUNT(DISTINCT fte.test_id) as impacted_tests,
	COUNT(CASE WHEN fte.status = 'flaky' THEN fte.id END) as flaky_tests,
	COUNT(CASE WHEN fte.status = 'skipped' THEN fte.id END) skipped,
	COUNT(CASE WHEN fte.status = 'blocklisted' THEN fte.id END) blocklisted,
	COUNT(CASE WHEN fte.status = 'aborted' THEN fte.id END) aborted,
	gc2.commit_id
FROM
	flaky_test_execution fte
JOIN build b ON
    b.id = fte.build_id	
JOIN git_commits gc2 ON
	gc2.commit_id = b.commit_id
JOIN repositories r2 ON
	gc2.repo_id = r2.id
JOIN discovery d ON
	d.commit_id = gc2.commit_id 
	AND d.test_id = fte.test_id 
WHERE
	r2.name = :repo
	AND r2.org_id = :org_id
	AND gc2.created_at >= :start_date
	AND gc2.created_at <= :end_date
	GROUP BY gc2.commit_id 
)
SELECT 
	COALESCE(ANY_VALUE(execution.impacted_tests),0) impacted,
	COALESCE(ANY_VALUE(execution.blocklisted),0) blocklisted,
	COALESCE(ANY_VALUE(execution.quarantined),0) quarantined,
	COUNT(discovery.test_id) total,
	COALESCE(fe.impacted_tests, 0) flaky_impacted_tests,
    COALESCE(fe.flaky_tests, 0) flaky_tests,
	COALESCE(fe.skipped, 0) flaky_skipped,
	COALESCE(fe.blocklisted, 0) flaky_blocklisted,
	COALESCE(fe.aborted, 0) flaky_aborted, 
	gc3.commit_id,
	ANY_VALUE(gc3.created_at) created_at
FROM
git_commits gc3
JOIN build b ON
	b.commit_id = gc3.commit_id
	AND b.repo_id = gc3.repo_id 
LEFT JOIN discovery ON
	gc3.commit_id = discovery.commit_id
LEFT JOIN execution ON 
	gc3.commit_id = execution.commit_id
LEFT JOIN flaky_execution fe ON 
	gc3.commit_id = fe.commit_id
JOIN repositories r3 ON
	gc3.repo_id = r3.id
	%s
WHERE 
	r3.name = :repo
AND r3.org_id = :org_id
AND gc3.created_at >= :start_date
AND gc3.created_at <= :end_date
AND b.status IN ('passed', 'failed', 'error')
	%s
GROUP BY gc3.commit_id ORDER BY created_at DESC
`
const findTestMetaQuery = `
WITH ranked_executions AS (
	SELECT
		te.test_id,
		te.status,
		ROW_NUMBER() OVER (PARTITION BY te.test_id ORDER BY te.updated_at DESC) rn
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
		%s
		%s
		%s
), flaky_ranked_executions AS (
	SELECT
		fte.test_id,
		fte.status,
		ROW_NUMBER() OVER (PARTITION BY fte.test_id ORDER BY fte.updated_at DESC) rn
	FROM
		flaky_test_execution fte
	JOIN test t ON
		t.id = fte.test_id
	JOIN build b ON
		b.id = fte.build_id
	JOIN repositories r ON
		r.id = t.repo_id
		%s
	WHERE
		r.name = :repo
		AND r.org_id = :org_id
		%s
		%s
		%s
), flaky_executions AS (
	SELECT
	*
	FROM
		flaky_ranked_executions
	WHERE
		rn = 1
)
SELECT
	COALESCE(COUNT(DISTINCT te.test_id),0) total,
	COALESCE(COUNT(DISTINCT CASE WHEN te.status = 'passed' THEN te.test_id END),0) passed,
	COALESCE(COUNT(DISTINCT CASE WHEN te.status = 'failed' THEN te.test_id END),0) failed,
	COALESCE(COUNT(DISTINCT CASE WHEN te.status = 'skipped' THEN te.test_id END),0) skipped,
	COALESCE(COUNT(DISTINCT CASE WHEN te.status = 'blocklisted' THEN te.test_id END),0) blocklisted,
	COALESCE(COUNT(DISTINCT CASE WHEN te.status = 'quarantined' THEN te.test_id END),0) quarantined,
	COALESCE(COUNT(DISTINCT CASE WHEN te.status = 'aborted' THEN te.test_id END),0) aborted,
	COALESCE(COUNT(DISTINCT CASE WHEN fe.status = 'flaky' THEN fe.test_id END),0) flaky,
	COALESCE(COUNT(DISTINCT CASE WHEN fe.status = 'nonflaky' THEN fe.test_id END),0) non_flaky
FROM
	ranked_executions te
LEFT JOIN flaky_executions fe ON
	fe.test_id = te.test_id 
WHERE
	te.rn = 1
`
const repoTestJobsQuery = `
SELECT
	t.id,
	t.name,
	t.test_suite_id,
	t.test_locator,
	JSON_ARRAYAGG(JSON_OBJECT('build_id',
	b.id,
	'status',
	te.status,
	'created_at',
	UNIX_TIMESTAMP(b.created_at),
	'start_time',
	UNIX_TIMESTAMP(b.start_time),
	'end_time',
	UNIX_TIMESTAMP(b.end_time),
	'branch', COALESCE(b.branch_name, ""))) status,
	MIN(t.created_at)
FROM
	test t
JOIN repositories r ON
	r.id = t.repo_id
JOIN build b ON
	b.repo_id  = r.id
LEFT JOIN test_execution te ON
	te.test_id = t.id
	AND te.build_id = b.id
WHERE 
	b.created_at >= ? 
	AND b.created_at <= ?
	AND r.name = ?
	AND r.org_id = ?	
`

const findBlocklistedTestsQuery = `
SELECT
	COUNT(DISTINCT t.id) AS Blocklisted_Tests
FROM
	test t
JOIN test_execution te ON
	t.id = te.test_id
JOIN git_commits gc ON
	gc.commit_id = te.commit_id
JOIN repositories r ON 
	gc.repo_id = r.id
	%s
WHERE
	te.status = 'blocklisted'
	AND r.name = :repo
	AND r.org_id = :org
	%s
	AND gc.author_name = :author
	AND gc.created_at >= :start_date
	AND gc.created_at <= :end_date
`

const findRepoIrregularTests = `
WITH search_space AS (
	SELECT
		te.test_id,
		te.duration,
		te.build_id,
		te.created_at executed_on,
		b.commit_id, 
		t.name,
		ROW_NUMBER() OVER(PARTITION BY te.test_id
	ORDER BY
		te.duration DESC, te.created_at DESC) AS descending,
		ROW_NUMBER() OVER(PARTITION BY te.test_id
	ORDER BY
		te.duration ASC, te.created_at DESC) AS ascending
	FROM
		test_execution te
	JOIN build b ON
		te.build_id = b.id
	JOIN test t ON
		t.id = te.test_id
	WHERE
		b.repo_id = :repo_id
		AND te.created_at <= :max_created_at
		AND te.created_at >= :min_created_at
		AND te.status IN ("passed", "failed")
		%s
	),
	filtered AS (
	SELECT
		ss.name,
		ss.build_id,
		ss.commit_id,
		ss.test_id,
		ss.duration,
		ss.executed_on
	FROM
		search_space ss
	WHERE
		ss.descending = 1
		OR ss.ascending = 1
	)
	SELECT
		f.test_id,
		f.name,
		f.build_id slow_build_id,
		f2.build_id fast_build_id,
		f.commit_id slow_commit_id,
		f2.commit_id fast_commit_id,
		f.executed_on slow_execution_time,
		f2.executed_on fast_execution_time,
		(f.duration) slow_duration,
		f2.duration fast_duration
	FROM
		filtered f
	JOIN filtered f2 ON
		f.duration > (1.5 * f2.duration)
		AND f.test_id = f2.test_id
	ORDER BY (f.duration - f2.duration) DESC
	LIMIT :limit
`

const findTestCountMonthWise = `
WITH ordered_commits_monthwise AS (
	SELECT
		cd.commit_id,
		cd.created_at,
		ROW_NUMBER() OVER(PARTITION BY MONTH(cd.created_at)
	ORDER BY
		cd.created_at DESC) as rn,
		JSON_LENGTH(cd.test_ids) total_tests
	FROM 
		commit_discovery cd
	%s
	WHERE
		cd.repo_id = :repo_id
		AND cd.created_at <= :max_created_at
		AND cd.created_at >= :min_created_at
		%s
	),
	latest_commits_monthwise AS (
	SELECT
		ocm.commit_id,
		ocm.total_tests,
		ocm.created_at
	FROM
		ordered_commits_monthwise ocm
	WHERE
		ocm.created_at >= :min_created_at
		AND ocm.created_at <= :max_created_at
		AND ocm.rn = 1
	ORDER BY
		ocm.created_at DESC
	)
	SELECT
		ANY_VALUE(lcm.total_tests) total_tests,
		ANY_VALUE(EXTRACT(MONTH FROM lcm.created_at)) month,
		ANY_VALUE(MONTHNAME(lcm.created_at)) month_name,
		ANY_VALUE(YEAR(lcm.created_at)) year
	FROM
		latest_commits_monthwise lcm
	GROUP BY
		lcm.commit_id
	ORDER BY
		year
`

const findAddedTests = `
WITH old_commit AS (
	SELECT
		gc.commit_id 
	FROM
		git_commits gc 
		%s
	WHERE
		gc.created_at < :min_created_at
		AND gc.repo_id = :repo_id
		%s
	ORDER BY gc.created_at DESC LIMIT 1
	),
	past_tests AS (
	SELECT
		DISTINCT(tt.test_id) test_ids,
		cd.commit_id
	FROM
		commit_discovery cd
	JOIN JSON_TABLE(
			cd.test_ids,
			'$[*]' COLUMNS(
				test_id varchar(32) PATH '$' ERROR ON
				ERROR
			)
		) tt
	JOIN old_commit oc ON
		oc.commit_id = cd.commit_id 
		%s
	WHERE
		cd.repo_id = :repo_id
		%s
	),
	current_tests AS (
	SELECT  
		DISTINCT(tt.test_id) test_ids
	FROM
		commit_discovery cd
	JOIN JSON_TABLE(
			cd.test_ids,
			'$[*]' COLUMNS(
				test_id varchar(32) PATH '$' ERROR ON
				ERROR
			)
		) tt
	%s
	WHERE
		cd.repo_id = :repo_id
		AND cd.created_at >= :min_created_at
		AND cd.created_at <= :max_created_at
		%s
	)
	SELECT
		COUNT(ct.test_ids) added_tests
	FROM
		current_tests ct
	LEFT JOIN past_tests pt ON
		ct.test_ids = pt.test_ids
	WHERE
		pt.test_ids IS NULL
`
