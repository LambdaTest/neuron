package core

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

// Branch represents a git branch.
type Branch struct {
	ID      string    `json:"-" db:"id"`
	Name    string    `json:"-" db:"name"`
	RepoID  string    `json:"-" db:"repo_id"`
	Created time.Time `json:"-" db:"created_at"`
	Updated time.Time `json:"-" db:"updated_at"`
}

// BranchStore defines datastore operation for working with branch.
type BranchStore interface {
	// CreateInTx persists a new branch in the datastore and executes the statements within the specified transaction
	CreateInTx(ctx context.Context, tx *sqlx.Tx, branch *Branch) error
	// FindByRepo fetch the list of all branches which are there for a particular user
	FindByRepo(ctx context.Context, repoID string) ([]string, error)
}
