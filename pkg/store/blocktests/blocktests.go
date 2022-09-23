package blocktests

import (
	"context"
	"fmt"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/jmoiron/sqlx"
)

type blocktestStore struct {
	db     core.DB
	logger lumber.Logger
}

const insertQueryChunkSize = 1000

// TODO: This API should return a list of test cases
// (file path, test suite name, and line number) that are blocklisted.
// The test case entity contains testlocator attribute which must be returned

// New returns a new BlockListStore
func New(db core.DB, logger lumber.Logger) core.BlockTestStore {
	return &blocktestStore{db: db, logger: logger}
}

func (b *blocktestStore) CreateInTx(ctx context.Context, tx *sqlx.Tx, blocktests []*core.BlockTest) error {
	return utils.Chunk(insertQueryChunkSize, len(blocktests), func(start int, end int) error {
		if _, err := tx.NamedExecContext(ctx, insertQuery, blocktests[start:end]); err != nil {
			return err
		}
		return nil
	})
}

func (b *blocktestStore) FindBlockTest(ctx context.Context, repoID, branch, statusFilter string) (blockTests []*core.BlockTest, err error) {
	blockTests = make([]*core.BlockTest, 0)
	return blockTests, b.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo_id": repoID,
			"branch":  branch,
			"status":  statusFilter,
		}
		typeWhere := "AND (b.status = 'blocklisted' OR b.status = 'quarantined')"
		if statusFilter != "" {
			typeWhere = "AND b.status = :status"
		}
		query := fmt.Sprintf(findBlockTestQuery, typeWhere)
		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}

		defer rows.Close()
		for rows.Next() {
			blockTest := new(core.BlockTest)
			if err = rows.Scan(&blockTest.TestName,
				&blockTest.Locator,
				&blockTest.Status); err != nil {
				return errs.SQLError(err)
			}
			blockTests = append(blockTests, blockTest)
		}
		return nil
	})
}

const insertQuery = `
INSERT
    INTO
    block_tests
    (id,
	test_id,
	status,
	repo_id,
	branch,
	blocked_by,
    created_at,
    updated_at)
VALUES (:id,
	:test_id,
	:status,
	:repo_id,
	:branch,
	:blocked_by,
	NOW(),
	NOW())
ON
DUPLICATE KEY
UPDATE
    status = 
	VALUES(status),
	updated_at =
	VALUES(updated_at)
`

const findBlockTestQuery = `
SELECT
	t.name,
	t.test_locator,
	b.status
FROM
	test t
JOIN block_tests b ON
	t.id = b.test_id	
WHERE
	b.repo_id=:repo_id
	AND b.branch=:branch
	%s
`
