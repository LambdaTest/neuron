package core

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

// TestBranch represents the test and branch relation
type TestBranch struct {
	ID         string    `json:"id,omitempty" db:"id"`
	TestID     string    `json:"test_id" db:"test_id"`
	RepoID     string    `json:"repo_id" db:"repo_id"`
	BranchName string    `json:"branch_name" db:"branch_name"`
	Updated    time.Time `json:"-" db:"updated_at"`
	Created    time.Time `json:"-" db:"created_at"`
}

// TestBranchStore defines datastore operation for working with test_branch store
type TestBranchStore interface {
	// CreateInTx persisits a new test and branch relation in the datastore and executes the statements within the specified transaction.
	CreateInTx(ctx context.Context, tx *sqlx.Tx, testBranch []*TestBranch) error
}
