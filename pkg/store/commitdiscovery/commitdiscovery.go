package commitdiscovery

import (
	"context"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/jmoiron/sqlx"
)

type commitDiscoveryStore struct {
	db     core.DB
	logger lumber.Logger
}

// New returns a new CommitDiscoveryStore.
func New(db core.DB, logger lumber.Logger) core.CommitDiscoveryStore {
	return &commitDiscoveryStore{db: db, logger: logger}
}

func (c *commitDiscoveryStore) CreateInTx(ctx context.Context, tx *sqlx.Tx, cd *core.CommitDiscovery) error {
	if _, err := tx.NamedExecContext(ctx, insertQuery, cd); err != nil {
		return errs.SQLError(err)
	}
	return nil
}

const insertQuery = `INSERT
INTO
commit_discovery(id,
repo_id,
commit_id,
test_ids,
submodule,
suite_ids)
VALUES (:id,
:repo_id,
:commit_id,
:test_ids,
:submodule,
:suite_ids) ON
DUPLICATE KEY
UPDATE
test_ids =
VALUES(test_ids),
suite_ids =
VALUES(suite_ids),
updated_at = NOW()`
