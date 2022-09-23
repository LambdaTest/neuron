package branchcommit

import (
	"context"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/jmoiron/sqlx"
)

type branchCommitStore struct {
	db     core.DB
	logger lumber.Logger
}

const insertQueryChunkSize = 1000

// New returns a new BranchCommitStore
func New(db core.DB, logger lumber.Logger) core.BranchCommitStore {
	return &branchCommitStore{db: db, logger: logger}
}

func (g *branchCommitStore) CreateInTx(ctx context.Context, tx *sqlx.Tx, branchCommits []*core.BranchCommit) error {
	return utils.Chunk(insertQueryChunkSize, len(branchCommits), func(start int, end int) error {
		if _, err := tx.NamedExecContext(ctx, insertQuery, branchCommits[start:end]); err != nil {
			return err
		}
		return nil
	})
}

const insertQuery = `INSERT
INTO
branch_commit(id,
commit_id,
branch_name,
repo_id,
created_at,
updated_at)
VALUES (:id,
:commit_id,
:branch_name,
:repo_id,
:created_at,
:updated_at)
ON
DUPLICATE KEY
UPDATE
updated_at =
VALUES(updated_at)`
