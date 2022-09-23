package core

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"gopkg.in/guregu/null.v4/zero"
)

// TestSuiteExecutionStatus specifies the status of a test suite which was executed
type TestSuiteExecutionStatus string

// TestSuiteExecution Status values
const (
	TestSuiteStopped     TestSuiteExecutionStatus = "stopped"
	TestSuiteFailed      TestSuiteExecutionStatus = "failed"
	TestSuitePassed      TestSuiteExecutionStatus = "passed"
	TestSuiteAborted     TestSuiteExecutionStatus = "aborted"
	TestSuiteSkipped     TestSuiteExecutionStatus = "skipped"
	TestSuitePending     TestSuiteExecutionStatus = "pending"
	TestSuiteBlocklisted TestSuiteExecutionStatus = "blocklisted"
)

// TestSuiteExecution represents the executed test suite
type TestSuiteExecution struct {
	ID              string                   `json:"id,omitempty" db:"id"`
	SuiteID         string                   `json:"suite_id,omitempty" db:"suite_id"`
	CommitID        string                   `json:"commit_id,omitempty" db:"commit_id"`
	Status          TestSuiteExecutionStatus `json:"status,omitempty" db:"status"`
	Created         time.Time                `json:"created_at" db:"created_at"`
	Updated         time.Time                `json:"-" db:"updated_at"`
	StartTime       zero.Time                `json:"start_time" db:"start_time"`
	EndTime         zero.Time                `json:"end_time" db:"end_time"`
	Duration        int                      `json:"duration" db:"duration"`
	Stdout          string                   `json:"-" db:"stdout"`
	Stderr          string                   `json:"-" db:"stderr"`
	BlocklistSource zero.String              `json:"-" db:"blocklist_source"`
	TaskID          string                   `json:"task_id,omitempty" db:"task_id"`
	BuildID         string                   `json:"build_id,omitempty" db:"build_id"`
	BuildNum        int                      `json:"build_num,omitempty"`
	BuildTag        string                   `json:"build_tag,omitempty"`
	Branch          string                   `json:"branch_name,omitempty"`
}

// TestSuiteExecutionStore defines datastore operation for working with test suite execution store
type TestSuiteExecutionStore interface {
	// CreateInTx persists a new test in the datastore and executes the statement in the specified transaction.
	CreateInTx(ctx context.Context, tx *sqlx.Tx, testSuiteExecution []*TestSuiteExecution) error
}
