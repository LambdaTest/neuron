package core

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"gopkg.in/guregu/null.v4/zero"
)

// FlakyConfigType defines type of flaky config
type FlakyConfigType string

// const defines all flaky config
const (
	FlakyConfigPostMerge FlakyConfigType = "postmerge"
	FlakyConfigPreMerge  FlakyConfigType = "premerge"
)

// FlakyAlgo is a type of Algo
type FlakyAlgo string

const (
	// RunningXTimes is a flaky algo
	RunningXTimes FlakyAlgo = "running_x_times"
)

// FlakyTestExecution represents flaky test execution results
type FlakyTestExecution struct {
	ID              string                       `db:"id" json:"-"`
	TestID          string                       `db:"test_id" json:"test_id,omitempty"`
	AlgoName        FlakyAlgo                    `db:"algo_name" json:"algo_name,omitempty"`
	ExecInfo        string                       `db:"exec_info" json:"-"`
	Status          TestExecutionStatus          `db:"status" json:"status,omitempty"`
	BuildID         string                       `db:"build_id" json:"build_id,omitempty"`
	TaskID          string                       `db:"task_id" json:"-"`
	CreatedAt       time.Time                    `db:"created_at" json:"-"`
	UpdatedAt       time.Time                    `db:"updated_at" json:"-"`
	TestName        string                       `json:"test_name,omitempty"`
	TestResults     []TestExecutionStatus        `json:"test_results,omitempty"`
	FlakyRate       float32                      `json:"flaky_rate"`
	ExecutionInfo   *AggregatedFlakyTestExecInfo `json:"-"`
	TestMeta        *FlakyExecutionMetadata      `json:"execution_meta,omitempty"`
	TestSuiteName   zero.String                  `json:"test_suite_name,omitempty"`
	TestSuiteID     zero.String                  `json:"test_suite_id,omitempty"`
	ExecutionStatus TestExecutionStatus          `json:"execution_status,omitempty"`
	FileName        string                       `json:"file_name,omitempty"`
}

// FlakyExecutionMetadata stores execution metadata for flaky_build
type FlakyExecutionMetadata struct {
	ImpactedTests      int       `json:"impacted_tests"`
	FlakyTests         int       `json:"flaky_tests"`
	OverAllFlakiness   float32   `json:"overall_flakiness"`
	Blocklisted        int       `json:"tests_blocklisted"`
	Skipped            int       `json:"tests_skipped"`
	Failed             int       `json:"tests_failed"`
	Aborted            int       `json:"tests_aborted"`
	Quarantined        int       `json:"tests_quarantined"`
	NonFlakyTests      int       `json:"tests_nonflaky"`
	JobsCount          int       `json:"jobs_count"`
	FirstFlakeID       string    `json:"first_flake_id"`
	FirstFlakeTime     time.Time `json:"first_flake_time"`
	LastFlakeID        string    `json:"last_flake_id"`
	LastFlakeTime      time.Time `json:"last_flake_time"`
	LastFlakeStatus    string    `json:"last_flake_status"`
	FirstFlakeCommitID string    `json:"first_flake_commit_id"`
	LastFlakeCommitID  string    `json:"last_flake_commit_id"`
}

// FlakyConfig defines flaky config
type FlakyConfig struct {
	Org             string          `json:"org,omitempty"`
	Repo            string          `json:"repo,omitempty"`
	ID              string          `db:"id" json:"id"`
	RepoID          string          `db:"repo_id" json:"repo_id"`
	Branch          string          `db:"branch" json:"branch"`
	IsActive        bool            `db:"is_active" json:"is_active"`
	AutoQuarantine  bool            `db:"auto_quarantine" json:"auto_quarantine"`
	ConfigType      FlakyConfigType `db:"config_type" json:"config_type"`
	AlgoName        FlakyAlgo       `db:"algo_name" json:"algo_name"`
	ConsecutiveRuns int             `db:"consecutive_runs" json:"consecutive_runs"`
	Threshold       int             `db:"threshold" json:"threshold"`
	CreatedAt       time.Time       `db:"created_at" json:"-"`
	UpdatedAt       time.Time       `db:"updated_at" json:"-"`
}

// AggregatedFlakyTestExecInfo stores aggerated info for flaky test results
type AggregatedFlakyTestExecInfo struct {
	Results            []TestExecutionStatus `json:"results"`
	Threshold          int                   `json:"threshold"`
	NumberOfExecutions int                   `json:"number_of_executions"`
	TotalTransitions   int                   `json:"total_transitions"`
	FirstTransition    int                   `json:"first_transition"`
	LastValidStatus    TestExecutionStatus   `json:"-"`
}

// FlakyTestStore defines datastore operation for working with Flakytest execution
type FlakyTestStore interface {
	// FindTests gives test details for all tests
	FindTests(ctx context.Context, buildID, taskID string,
		statusFilter, executionStatusFilter, searchText string, offset, limit int) ([]*FlakyTestExecution, error)
	// CreateInTx persists a new flaky test in the datastore.
	CreateInTx(ctx context.Context, tx *sqlx.Tx, FlakyTestExecutionResults []*FlakyTestExecution) error
	// MarkTestsToStableInTx marks quarantined tests to non flaky
	MarkTestsToStableInTx(ctx context.Context, tx *sqlx.Tx, buildID string) (int64, error)
	// FindTestsInJob gives the total flaky tests in a particular job.
	FindTestsInJob(ctx context.Context, repoName, orgID, branchName string,
		startDate, endDate time.Time) ([]*FlakyTestExecution, error)
	// ListFlakyTests return the total flaky tests in a particular job.
	ListFlakyTests(ctx context.Context, repoName, orgID, branchName string,
		startDate, endDate time.Time, offset, limit int) ([]*FlakyTestExecution, error)
}

// FlakyConfigStore has datastore operations contracts for flakyconfig table
type FlakyConfigStore interface {
	// Create persists a new postmergeconfig in the datastore
	Create(ctx context.Context, flakyConfig *FlakyConfig) error
	// Update updates existing flakyconfig in the datastore
	Update(ctx context.Context, flakyConfig *FlakyConfig) error
	// FindIfActiveConfigExists checks if there is an active config for given branch, repo and config type
	FindIfActiveConfigExists(ctx context.Context,
		repoID, branchName string, configType FlakyConfigType) (err error)
	// FindActiveFlakyConfig returns active flakyconfig by repoid and branch name
	FindActiveFlakyConfig(ctx context.Context, repoID string,
		branchName string, configType FlakyConfigType) (flakyConfig *FlakyConfig, err error)
	// FindAllFlakyConfig returns all flakyconfig by repoID
	FindAllFlakyConfig(ctx context.Context, repoID string) (flakyConfigList []*FlakyConfig, sqlErr error)
}
