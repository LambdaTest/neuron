package core

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

// TestSuiteBranch represents the test_suite and branch relation.
type TestSuiteBranch struct {
	ID          string    `json:"id,omitempty" db:"id"`
	TestSuiteID string    `json:"test_suite_id" db:"test_suite_id"`
	RepoID      string    `json:"repo_id" db:"repo_id"`
	BranchName  string    `json:"branch_name" db:"branch_name"`
	Updated     time.Time `json:"-" db:"updated_at"`
	Created     time.Time `json:"-" db:"created_at"`
}

// TestSuiteBranchStore defines datastore operation for working with test_suite_branch store.
type TestSuiteBranchStore interface {
	// CreateInTx persisits a new test_suite and branch relation in the datastore and executes the statements within the specified transaction.
	CreateInTx(ctx context.Context, tx *sqlx.Tx, tb []*TestSuiteBranch) error
}
