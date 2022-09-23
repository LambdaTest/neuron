package testsuiteexecution

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

type testSuiteExecutionStore struct {
	db     core.DB
	logger lumber.Logger
}

const insertQueryChunkSize = 1000

// New return new TestSuiteExecutionStore
func New(db core.DB, logger lumber.Logger) core.TestSuiteExecutionStore {
	return &testSuiteExecutionStore{db: db, logger: logger}
}

func (t *testSuiteExecutionStore) CreateInTx(ctx context.Context, tx *sqlx.Tx, testSuiteExecutionData []*core.TestSuiteExecution) error {
	return utils.Chunk(insertQueryChunkSize, len(testSuiteExecutionData), func(start int, end int) error {
		args := []interface{}{}
		placeholderGrps := []string{}
		for _, execData := range testSuiteExecutionData[start:end] {
			placeholderGrps = append(placeholderGrps, "(?,?,?,?,?,?,?,?,?,?)")
			args = append(args, execData.ID, execData.SuiteID, execData.CommitID, execData.Status, execData.StartTime, execData.EndTime,
				execData.Duration, execData.BlocklistSource, execData.BuildID, execData.TaskID)
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
    test_suite_execution (
        id,
        suite_id,
        commit_id,
        status,
        start_time,
        end_time,
        duration,
        blocklist_source,
        build_id,
        task_id
    )
VALUES %s
`
