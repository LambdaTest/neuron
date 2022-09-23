package core

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"gopkg.in/guregu/null.v4/zero"
)

// TestSuite represents the user's test_suites.
type TestSuite struct {
	ID            string              `db:"id" json:"id"`
	ParentSuiteID zero.String         `db:"parent_suite_id" json:"parent_suite_id"`
	Name          string              `db:"name" json:"name"`
	FilePath      zero.String         `db:"file_path" json:"-"`
	DebutCommit   string              `db:"debut_commit" json:"debut_commit"`
	RepoID        string              `db:"repo_id" json:"-"`
	Created       time.Time           `db:"created_at" json:"-"`
	Updated       time.Time           `db:"updated_at" json:"-"`
	Meta          *SuiteExecutionMeta `json:"execution_meta,omitempty"`
	LatestBuild   *Build              `json:"latest_build,omitempty"`
	TotalTests    int                 `db:"total_tests" json:"total_tests"`
	Execution     *TestSuiteExecution `json:"execution_details,omitempty"`
	SubModule     string              `db:"submodule" json:"submodule"`
	Branch        string              `json:"branch,omitempty"`
}

// SuiteExecutionMeta contains additional info of the test suites executed
type SuiteExecutionMeta struct {
	Total                int        `json:"total_executions"`
	AvgTestSuiteDuration zero.Float `json:"avg_test_suite_duration"`
	Passed               int        `json:"tests_suite_passed"`
	Blocklisted          int        `json:"tests_suite_blocklisted"`
	Skipped              int        `json:"tests_suite_skipped"`
	Failed               int        `json:"tests_suite_failed"`
	Aborted              int        `json:"tests_suite_aborted"`
}

// TestSuiteStore defines datastore operation for working with test suites
type TestSuiteStore interface {
	// CreateInTx persists a new test suite in the datastore and executes the statement in the specified transaction.
	CreateInTx(ctx context.Context, tx *sqlx.Tx, testData []*TestSuite) error
	// FindByRepo returns the test suites for a repository.
	FindByRepo(ctx context.Context, repoName, orgID, suiteID, branchName, statusFilter,
		searchText string, authorsNames []string, offset, limit int) ([]*TestSuite, error)
	// FindExecution returns the execution data for the test suites.
	FindExecution(ctx context.Context, testSuiteID, repoName, orgID, branchName, statusFilter, searchText string,
		startDate, endDate time.Time, offset, limit int) ([]*TestSuiteExecution, error)
	// FindStatus returns test execution status of the test suites.
	FindStatus(ctx context.Context, testSuiteID, repoName, orgID, branchName string,
		startDate, endDate time.Time) ([]*TestSuiteExecution, error)
	// FindMeta returns suite meta of all the test suites.
	FindMeta(ctx context.Context, repoName, orgID, buildID, commitID,
		branchName string) ([]*SuiteExecutionMeta, error)
}
