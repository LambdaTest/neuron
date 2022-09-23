package flakyexecution

import (
	"context"
	"time"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/jmoiron/sqlx"
)

const (
	maxRetries = 3
	delay      = 200 * time.Millisecond
	maxJitter  = 100 * time.Millisecond
	errMsg     = "failed to perform executions transaction"
)

// flakyExecutionStore executes all the queries needed in test and suite listing in a transaction
type flakyExecutionStore struct {
	db             core.DB
	flakyTestStore core.FlakyTestStore
	blockTestStore core.BlockTestStore
	logger         lumber.Logger
}

// New returns a new ExecutionStore
func New(db core.DB,
	flakyTestStore core.FlakyTestStore,
	blockTestStore core.BlockTestStore,
	logger lumber.Logger,
) core.FlakyExecutionStore {
	return &flakyExecutionStore{
		db:             db,
		flakyTestStore: flakyTestStore,
		blockTestStore: blockTestStore,
		logger:         logger,
	}
}

func (fes *flakyExecutionStore) Create(ctx context.Context,
	flakyExecutionResults []*core.FlakyTestExecution,
	blockTests []*core.BlockTest,
	buildID string,
) error {
	return fes.db.ExecuteTransactionWithRetry(ctx, maxRetries, delay, maxJitter, errMsg,
		func(tx *sqlx.Tx) error {
			if len(flakyExecutionResults) > 0 {
				if err := fes.flakyTestStore.CreateInTx(ctx, tx, flakyExecutionResults); err != nil {
					fes.logger.Errorf("failed to insert in flaky test execution table buildID %s, error %v", buildID, err)
					return err
				}

				rowCount, err := fes.flakyTestStore.MarkTestsToStableInTx(ctx, tx, buildID)
				if err != nil {
					fes.logger.Errorf("failed to remove stable tests from blocklist store buildID %s, error %v", buildID, err)
					return err
				}
				fes.logger.Debugf("Number of tests transitioned from quarantined to stable in current build %d", rowCount)
			}

			if len(blockTests) > 0 {
				if err := fes.blockTestStore.CreateInTx(ctx, tx, blockTests); err != nil {
					fes.logger.Errorf("failed to insert in blocktest in table buildID %s, error %v", buildID, err)
					return err
				}
			}

			return nil
		})
}
