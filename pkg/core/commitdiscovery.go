package core

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"
)

// CommitDiscovery represents the tests and suites discovered in a commit.
type CommitDiscovery struct {
	ID        string          `db:"id"`
	RepoID    string          `db:"repo_id"`
	CommitID  string          `db:"commit_id"`
	TestIDs   json.RawMessage `db:"test_ids"`
	SuiteIDs  json.RawMessage `db:"suite_ids"`
	Created   time.Time       `db:"created_at"`
	Updated   time.Time       `db:"updated_at"`
	SubModule string          `db:"submodule"`
}

// CommitDiscoveryStore defines datastore operation for working with commit_discovery store.
type CommitDiscoveryStore interface {
	// CreateInTx persists a the tests and suites discovered for a commit in the datastore and
	// executes the statements within the specified transaction
	CreateInTx(ctx context.Context, tx *sqlx.Tx, entity *CommitDiscovery) error
}
