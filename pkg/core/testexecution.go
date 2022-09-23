package core

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"gopkg.in/guregu/null.v4/zero"
)

// TestExecutionStatus specifies the status of a test which was executed
type TestExecutionStatus string

// TestExecution Status values.
const (
	TestFailed      TestExecutionStatus = "failed"
	TestPassed      TestExecutionStatus = "passed"
	TestAborted     TestExecutionStatus = "aborted"
	TestSkipped     TestExecutionStatus = "skipped"
	TestPending     TestExecutionStatus = "pending"
	TestBlocklisted TestExecutionStatus = "blocklisted"
	TestQuarantined TestExecutionStatus = "quarantined"
	TestFlaky       TestExecutionStatus = "flaky"
	TestNonFlaky    TestExecutionStatus = "nonflaky"
	TestNotRun      TestExecutionStatus = "notrun"
)

// TestStatusWeight returns the weight of the test status
const (
	TestFailedWeight = iota + 1
	TestAbortedWeight
	TestPassedWeight
	TestSkippedWeight
	TestDefaultWeight
)

// Weight returns the weight of the test status
func (t TestExecutionStatus) Weight() int {
	switch t {
	case TestFailed:
		return TestFailedWeight
	case TestPassed:
		return TestPassedWeight
	case TestAborted:
		return TestAbortedWeight
	case TestBlocklisted, TestSkipped, TestPending, TestFlaky, TestQuarantined, TestNonFlaky:
		return TestSkippedWeight
	default:
		return TestDefaultWeight
	}
}

// TestExecution represents the executed  test.
type TestExecution struct {
	ID              string              `json:"id,omitempty" db:"id"`
	TestID          string              `json:"test_id,omitempty" db:"test_id"`
	CommitID        string              `json:"commit_id,omitempty" db:"commit_id"`
	CommitAuthor    string              `json:"commit_author,omitempty"`
	CommitMessage   *zero.String        `json:"commit_message,omitempty"`
	Status          TestExecutionStatus `json:"status,omitempty" db:"status"`
	Created         time.Time           `json:"created_at" db:"created_at"`
	Updated         time.Time           `json:"-" db:"updated_at"`
	StartTime       zero.Time           `json:"start_time" db:"start_time"`
	EndTime         zero.Time           `json:"end_time" db:"end_time"`
	Duration        int                 `json:"duration" db:"duration"`
	Stdout          string              `json:"-" db:"stdout"`
	Stderr          string              `json:"-" db:"stderr"`
	BlocklistSource zero.String         `json:"-" db:"blocklist_source"`
	TaskID          string              `json:"task_id,omitempty" db:"task_id"`
	BuildID         string              `json:"build_id,omitempty" db:"build_id"`
	BuildNum        int                 `json:"build_num,omitempty"`
	TestName        string              `json:"test_name,omitempty"`
	TestSuiteName   zero.String         `json:"test_suite_name,omitempty"`
	SuiteID         zero.String         `json:"test_suite_id,omitempty"`
	BuildTag        string              `json:"build_tag,omitempty"`
	Transition      *Transition         `json:"transition,omitempty"`
	Branch          string              `json:"branch,omitempty"`
}

// ExecutionMeta contains additional info of the tests executed
type ExecutionMeta struct {
	Total             int     `json:"total_tests_executed"`
	Passed            int     `json:"tests_passed"`
	Blocklisted       int     `json:"tests_blocklisted"`
	Quarantined       int     `json:"tests_quarantined"`
	Skipped           int     `json:"tests_skipped"`
	Failed            int     `json:"tests_failed"`
	Aborted           int     `json:"tests_aborted"`
	TotalTestDuration int     `json:"total_test_duration"`
	AvgTestDuration   float64 `json:"avg_test_duration"`
	TotalTests        int     `json:"total_tests"`
	Unimpacted        int     `json:"tests_unimpacted"`
	Completed         int     `json:"tests_completed"`
	Flaky             int     `json:"flaky_tests"`
	NonFlaky          int     `json:"non_flaky_tests"`
}

// TestMetrics represents the metrics for each test.
type TestMetrics struct {
	ID              string    `json:"-"`
	TestExecutionID string    `json:"-"`
	Memory          uint64    `json:"memory"`
	CPU             float64   `json:"cpu"`
	Storage         uint64    `json:"storage"`
	RecordTime      time.Time `json:"record_time"`
}

// TestExecutionStore defines datastore operation for working with test execution store
type TestExecutionStore interface {
	// CreateInTx persists a new test in the datastore and executes the statement in the specified transaction.
	CreateInTx(ctx context.Context, tx *sqlx.Tx, testExecution []*TestExecution) error
	// FindByTestID finds the executions of a test.
	FindByTestID(ctx context.Context, testID, repoName, orgID, branchName, statusFilter, searchID string,
		startDate, endDate time.Time, offset, limit int) ([]*TestExecution, error)
	// FindStatus finds the status of the test.
	FindStatus(ctx context.Context, testID, interval, repoName, orgID, branchName string,
		startDate, endDate time.Time) ([]*TestExecution, error)
	// FindExecutionTimeImpactedTests finds the total time taken by all the impacted tests in there execution
	FindExecutionTimeImpactedTests(ctx context.Context, commitID, buildID, repoID string) (totalImpactedTests, totalTime int, err error)
	// FindTimeByRunningAllTests finds the total time taken by all the tests discovered in the buildID, commitID
	FindTimeByRunningAllTests(ctx context.Context, buildID, commitID, repoID string) (totalTests, totalTime int, err error)
	// FindRepoTestStatusesFailed returns the test execution statuses for a repository.
	FindRepoTestStatusesFailed(ctx context.Context, repoName, orgID, branchName string,
		startDate, endDate time.Time, limit int) ([]*Test, error)
	// FindLatestExecution finds the latest execution for given testIDs and repo.
	FindLatestExecution(ctx context.Context, repoID string, testIDs []string) ([]*Test, error)

	// FindImpactedTestsByBuild returns list of all impacted test for given build
	FindImpactedTestsByBuild(ctx context.Context, buildID string) (map[string][]*Test, error)
}

// TestExecutionService defines operations for working with test execution data
type TestExecutionService interface {
	// StoreTestMetrics stores test metrics on azure blob in csv format.
	StoreTestMetrics(ctx context.Context, path string, metrics [][]string) error
	// FetchMetrics fetches the metrics from azure blob in array format.
	FetchMetrics(ctx context.Context, path, executionID string) (metrics []*TestMetrics, err error)
	// StoreTestFailures stores test failures on azure blob in json format.
	StoreTestFailures(ctx context.Context, path string, failureDetails map[string]string) error
	// FetchTestFailures fetches the test failures from azure blob in array format.
	FetchTestFailures(ctx context.Context, path, executionID string) (failureMsg string, err error)
}
