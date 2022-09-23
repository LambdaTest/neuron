package testcoverage

import (
	"context"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/jmoiron/sqlx"
)

type testcoverageStore struct {
	db     core.DB
	logger lumber.Logger
}

const insertQueryChunkSize = 1000

// New returns a new TestCoverageStore.
func New(db core.DB, logger lumber.Logger) core.TestCoverageStore {
	return &testcoverageStore{db: db, logger: logger}
}

//todo: add build_id
func (tc *testcoverageStore) CreateInTx(ctx context.Context, tx *sqlx.Tx, coverage []*core.TestCoverage) error {
	return utils.Chunk(insertQueryChunkSize, len(coverage), func(start int, end int) error {
		if _, err := tx.NamedExecContext(ctx, insertQuery, coverage[start:end]); err != nil {
			return err
		}
		return nil
	})
}

func (tc *testcoverageStore) FindCommitCoverage(ctx context.Context, repoID, commitID string) (*core.TestCoverage, error) {
	coverage := new(core.TestCoverage)
	return coverage, tc.db.Execute(func(db *sqlx.DB) error {
		result := db.QueryRowxContext(ctx, selectQuery, repoID, commitID)
		if err := result.StructScan(coverage); err != nil {
			return errs.SQLError(err)
		}
		return nil
	})
}

const insertQuery = `INSERT
INTO
test_coverage(
id,
commit_id,
repo_id,
blob_link,
total_coverage,
created_at,
updated_at)
VALUES (:id,
:commit_id,
:repo_id,
:blob_link,
:total_coverage,
:created_at,
:updated_at) ON
DUPLICATE KEY
UPDATE
total_coverage =
VALUES(total_coverage),
blob_link =
VALUES(blob_link),
updated_at =
VALUES(updated_at)`

const selectQuery = `
SELECT
	repo_id,
	commit_id,
	total_coverage,
	blob_link
FROM
	test_coverage
WHERE
	repo_id = ?
	AND commit_id = ?
`
