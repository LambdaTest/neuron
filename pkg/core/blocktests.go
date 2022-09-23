package core

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

// BlockTest represents the blocklisted test
type BlockTest struct {
	ID        string              `db:"id"`
	TestID    string              `db:"test_id"`
	Status    TestExecutionStatus `db:"status" json:"status"`
	BlockedBy string              `db:"blocked_by"`
	RepoID    string              `db:"repo_id"`
	Branch    string              `db:"branch"`
	Created   time.Time           `db:"created_at"`
	Updated   time.Time           `db:"updated_at"`
	Locator   string              `json:"test_locator"`
	TestName  string              `json:"test_name"`
}

// BlockTestStore defines operations for working with blocktest store
type BlockTestStore interface {
	// CreateInTx persists a new blocklist in the datastore within transaction
	CreateInTx(ctx context.Context, tx *sqlx.Tx, blockTests []*BlockTest) error
	// FindBlockTest returns block tests
	FindBlockTest(ctx context.Context, repoID, branch, typeFilter string) (blockTests []*BlockTest, err error)
}
