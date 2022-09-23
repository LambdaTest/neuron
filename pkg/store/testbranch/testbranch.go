package testbranch

import (
	"context"
	"fmt"
	"strings"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/gocraft/dbr"
	"github.com/gocraft/dbr/dialect"
	"github.com/jmoiron/sqlx"
)

type testBranchStore struct {
	db     core.DB
	logger lumber.Logger
}

const insertQueryChunkSize = 1000

// New returns a new TestBranchStore.
func New(db core.DB, logger lumber.Logger) core.TestBranchStore {
	return &testBranchStore{db: db, logger: logger}
}

func (t *testBranchStore) CreateInTx(ctx context.Context, tx *sqlx.Tx, testBranch []*core.TestBranch) error {
	return utils.Chunk(insertQueryChunkSize, len(testBranch), func(start int, end int) error {
		args := []interface{}{}
		placeholderGrps := []string{}
		for _, tb := range testBranch[start:end] {
			placeholderGrps = append(placeholderGrps, "(?,?,?,?,?,?)")
			args = append(args, tb.ID, tb.TestID, tb.RepoID, tb.BranchName, tb.Created, tb.Updated)
		}
		interpolatedQuery, errI := dbr.InterpolateForDialect(fmt.Sprintf(insertQuery, strings.Join(placeholderGrps, ",")), args, dialect.MySQL)
		if errI != nil {
			return errs.SQLError(errI)
		}
		if _, err := tx.ExecContext(ctx, interpolatedQuery); err != nil {
			return errs.SQLError(err)
		}
		return nil
	})
}

const insertQuery = `
INSERT
	INTO
	test_branch(
		id,
		test_id,
		repo_id,
		branch_name,
		created_at,
		updated_at
	)
VALUES %s ON
DUPLICATE KEY
UPDATE
	updated_at =
VALUES(updated_at)
`
