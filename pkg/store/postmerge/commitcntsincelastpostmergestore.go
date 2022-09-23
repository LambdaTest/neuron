package postmerge

import (
	"context"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/jmoiron/sqlx"
)

type commitCntSinceLastPostMergeStore struct {
	db     core.DB
	logger lumber.Logger
}

// NewCommitCntSinceLastPostMergeStore returns a new instance.
func NewCommitCntSinceLastPostMergeStore(db core.DB, logger lumber.Logger) core.CommitCntSinceLastPostMergeStore {
	return &commitCntSinceLastPostMergeStore{db: db, logger: logger}
}

func (cs *commitCntSinceLastPostMergeStore) CreateInTx(ctx context.Context, tx *sqlx.Tx,
	commitCnt *core.CommitCntSinceLastPostMerge) error {
	if _, err := tx.NamedExecContext(ctx, insertCommitCntQuery, commitCnt); err != nil {
		return errs.SQLError(err)
	}
	return nil
}

func (cs *commitCntSinceLastPostMergeStore) FindInTx(ctx context.Context, tx *sqlx.Tx,
	repoID, branchName string) (*core.CommitCntSinceLastPostMerge, error) {
	commitCnt := new(core.CommitCntSinceLastPostMerge)
	row := tx.QueryRowxContext(ctx, findCommitCntQuery, repoID, branchName)
	if err := row.StructScan(commitCnt); err != nil {
		return commitCnt, errs.SQLError(err)
	}
	return commitCnt, nil
}

const insertCommitCntQuery = `INSERT
INTO
commit_cnt_since_last_postmerge(id,
repo_id,
branch,
commit_cnt,
created_at,
updated_at)
VALUES (:id,
:repo_id,
:branch,	
:commit_cnt,
:created_at,
:updated_at)
ON
DUPLICATE KEY
UPDATE
commit_cnt = 
VALUES(commit_cnt),
updated_at =
VALUES(updated_at)`

const findCommitCntQuery = `SELECT
id,
repo_id,
branch,
commit_cnt,
created_at,
updated_at
FROM
    commit_cnt_since_last_postmerge 
WHERE
	repo_id=?
    And
	branch=?`
