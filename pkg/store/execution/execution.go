package execution

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

// executionStore executes all the queries needed in test and suite listing in a transaction
type executionStore struct {
	db                      core.DB
	testExecutionStore      core.TestExecutionStore
	testSuiteExecutionStore core.TestSuiteExecutionStore
	testStore               core.TestStore
	testSuiteStore          core.TestSuiteStore
	testExecutionService    core.TestExecutionService
	logger                  lumber.Logger
}

// New returns a new ExecutionStore
func New(db core.DB,
	testExecutionStore core.TestExecutionStore,
	testSuiteExecutionStore core.TestSuiteExecutionStore,
	testStore core.TestStore,
	testSuiteStore core.TestSuiteStore,
	testExecutionService core.TestExecutionService,
	logger lumber.Logger,
) core.ExecutionStore {
	return &executionStore{
		db:                      db,
		testExecutionStore:      testExecutionStore,
		testSuiteExecutionStore: testSuiteExecutionStore,
		testStore:               testStore,
		testSuiteStore:          testSuiteStore,
		testExecutionService:    testExecutionService,
		logger:                  logger,
	}
}

func (es *executionStore) Create(ctx context.Context,
	testExecutions []*core.TestExecution,
	testSuiteExecutions []*core.TestSuiteExecution,
	buildID string,
) error {
	return es.db.ExecuteTransactionWithRetry(ctx, maxRetries, delay, maxJitter, errMsg,
		func(tx *sqlx.Tx) error {
			if len(testSuiteExecutions) > 0 {
				if err := es.testSuiteExecutionStore.CreateInTx(ctx, tx, testSuiteExecutions); err != nil {
					es.logger.Errorf("failed to insert test suites executions, buildID %s, error: %v", buildID, err)
					return err
				}
			}

			if len(testExecutions) > 0 {
				if err := es.testExecutionStore.CreateInTx(ctx, tx, testExecutions); err != nil {
					es.logger.Errorf("failed to insert tests executions, buildID %s, error: %v", buildID, err)
					return err
				}
			}
			return nil
		})
}
