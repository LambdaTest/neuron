package core

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"gopkg.in/guregu/null.v4/zero"
)

// Test represents the user's tests.
type Test struct {
	ID            string                  `db:"id" json:"id"`
	Name          string                  `db:"name" json:"name"`
	TestSuiteID   zero.String             `db:"test_suite_id" json:"test_suite_id,omitempty"`
	DebutCommit   string                  `db:"debut_commit" json:"debut_commit,omitempty"`
	RepoID        string                  `db:"repo_id" json:"repo_id,omitempty"`
	TestLocator   string                  `db:"test_locator" json:"test_locator,omitempty"`
	Status        zero.String             `db:"status" json:"-"`
	Execution     *TestExecution          `json:"execution_details,omitempty"`
	Meta          *ExecutionMeta          `json:"execution_meta,omitempty"`
	FlakyMeta     *FlakyExecutionMetadata `json:"flaky_execution_meta,omitempty"`
	LatestBuild   *Build                  `json:"latest_build,omitempty"`
	Suite         *TestSuite              `json:"suite,omitempty"`
	Created       time.Time               `db:"created_at" json:"created_at"`
	Updated       time.Time               `db:"updated_at" json:"-"`
	TestSuiteName zero.String             `json:"test_suite_name,omitempty"`
	TestStatus    []*TestStatus           `json:"test_status,omitempty"`
	Transition    *Transition             `json:"transition,omitempty"`
	Introduced    *time.Time              `json:"introduced_at,omitempty"`
	SubModule     string                  `db:"submodule" json:"submodule"`
}

// IrregularTests represents the tests whose execution time differs by more than 1.5 times
type IrregularTests struct {
	TestID            string    `json:"test_id" db:"test_id"`
	TestName          string    `json:"test_name" db:"name"`
	SlowBuildID       string    `json:"slow_build_id" db:"slow_build_id"`
	FastBuildID       string    `json:"fast_build_id" db:"fast_build_id"`
	SlowCommitID      string    `json:"slow_commit_id" db:"slow_commit_id"`
	FastCommitID      string    `json:"fast_commit_id" db:"fast_commit_id"`
	SlowExecutionTime time.Time `json:"slow_execution_time" db:"slow_execution_time"`
	FastExecutionTime time.Time `json:"fast_execution_time" db:"fast_execution_time"`
	SlowDuration      uint64    `json:"slow_duration" db:"slow_duration"`
	FastDuration      uint64    `json:"fast_duration" db:"fast_duration"`
}

// MonthWiseNetTests represents tests found for a repo in a month
type MonthWiseNetTests struct {
	MonthNo    int    `json:"month" db:"month"`
	MonthName  string `json:"month_name" db:"month_name"`
	Year       int    `json:"year" db:"year"`
	TotalTests uint64 `json:"total_tests" db:"total_tests"`
}

// Transition represents a test's transition from Passed to Failed, Failed to Blocklisted etc
type Transition struct {
	PreviousStatus string `json:"test_previous_status"`
	CurrentStatus  string `json:"test_current_status"`
}

// TestStore defines datastore operation for working with test
type TestStore interface {
	// CreateInTx persists a new test in the datastore and executes the statement in the specified transaction.
	CreateInTx(ctx context.Context, tx *sqlx.Tx, testData []*Test) error
	// FindByRepo returns the tests for a repository.
	FindByRepo(ctx context.Context, repoName, orgID, testID, branchName, statusFilter, searchText string, authorsNames []string,
		offset, limit int) ([]*Test, error)
	// FindRepoTestStatuses returns the test execution statuses for a repository.
	FindRepoTestStatuses(ctx context.Context, repoName, orgID, branchName, status, tag, searchText, lastSeenID string,
		startDate, endDate time.Time, limit int, isJobs bool) ([]*Test, error)
	// FindUninpactedTests returns the test which are not impacted by the given build and commit.
	FindUnimpactedTests(ctx context.Context, commitID, buildID, repoName, orgID string, offset, limit int) ([]*Test, error)
	// FindIntroducedDate returns the first date when this test was introduced
	FindIntroducedDate(ctx context.Context, repoName, orgID, branchName, lastSeenID string, limit int) (map[string]time.Time, error)
	// FindRepoTestStatusesSlowest returns the test execution statuses for a repository.
	FindRepoTestStatusesSlowest(ctx context.Context, repoName, orgID, branchName string,
		startDate, endDate time.Time, limit int) ([]*Test, error)
	// FindTestData return the tests information in a selected date range with respect to commit id.
	FindTestData(ctx context.Context, repoName, orgID, branchName string, startDate, endDate time.Time) ([]*GitCommit, error)
	// FindTestMeta returns the tests meta for all the tests in a selected repo
	FindTestMeta(ctx context.Context, repoName, orgID, buildID string,
		commitID, branchName string) ([]*ExecutionMeta, error)
	// FindBlocklistedTests returns the number of blocklisted tests by an author
	FindBlocklistedTests(ctx context.Context, repoName, orgID, branchName, authorName string, startDate, endDate time.Time) (int, error)
	// FindIrregularTests returns the tests which shows irregularity in execution time
	FindIrregularTests(ctx context.Context, repoID, branch string, startDate, endDate time.Time, limit int) ([]*IrregularTests, error)
	// FindMonthWiseNetTests returns monthwise total tests present in the repo between start-date and end-date
	FindMonthWiseNetTests(ctx context.Context, RepoID, branchName string, startDate, endDate time.Time) ([]*MonthWiseNetTests, error)
	// FindAddedTests returns the total tests added in a repo in a given duration
	FindAddedTests(ctx context.Context, repoID, branchName string, startDate,
		endDate time.Time) (int64, error)
}
