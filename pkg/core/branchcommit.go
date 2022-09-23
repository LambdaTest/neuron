package core

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

// BranchCommit represents a git branch.
type BranchCommit struct {
	ID         string    `json:"-" db:"id"`
	CommitID   string    `json:"-" db:"commit_id"`
	BranchName string    `json:"-" db:"branch_name"`
	RepoID     string    `json:"-" db:"repo_id"`
	Created    time.Time `json:"-" db:"created_at"`
	Updated    time.Time `json:"-" db:"updated_at"`
}

// BranchCommitStore defines datastore operation for working with branch_commit relationship table.
type BranchCommitStore interface {
	// CreateInTx persists a new entries to the branch_commit relationship table.
	CreateInTx(ctx context.Context, tx *sqlx.Tx, branchCommits []*BranchCommit) error
}
