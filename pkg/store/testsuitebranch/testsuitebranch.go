package testsuitebranch

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

type testSuiteBranchStore struct {
	db     core.DB
	logger lumber.Logger
}

const insertQueryChunkSize = 1000

// New returns a new TestSuiteBranchStore.
func New(db core.DB, logger lumber.Logger) core.TestSuiteBranchStore {
	return &testSuiteBranchStore{db: db, logger: logger}
}

func (t *testSuiteBranchStore) CreateInTx(ctx context.Context, tx *sqlx.Tx, testSuiteBranch []*core.TestSuiteBranch) error {
	return utils.Chunk(insertQueryChunkSize, len(testSuiteBranch), func(start int, end int) error {
		args := []interface{}{}
		placeholderGrps := []string{}
		for _, tsb := range testSuiteBranch[start:end] {
			placeholderGrps = append(placeholderGrps, "(?,?,?,?,?,?)")
			args = append(args, tsb.ID, tsb.TestSuiteID, tsb.RepoID, tsb.BranchName, tsb.Created, tsb.Updated)
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
	test_suite_branch(
		id,
		test_suite_id,
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
