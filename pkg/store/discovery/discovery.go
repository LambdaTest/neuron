package discovery

import (
	"context"
	"time"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/jmoiron/sqlx"
)

// discoveryStore executes all the queries needed in test and suite listing in a transaction
type discoveryStore struct {
	db                   core.DB
	testStore            core.TestStore
	testSuiteStore       core.TestSuiteStore
	testBranchStore      core.TestBranchStore
	testSuiteBranchStore core.TestSuiteBranchStore
	commitDiscoveryStore core.CommitDiscoveryStore
	logger               lumber.Logger
}

const (
	maxRetries = 3
	delay      = 250 * time.Millisecond
	maxJitter  = 100 * time.Millisecond
	errMsg     = "failed to perform discovery transaction"
)

// New returns a new DiscoveryStore
func New(db core.DB,
	testStore core.TestStore,
	testSuiteStore core.TestSuiteStore,
	testBranchStore core.TestBranchStore,
	testSuiteBranchStore core.TestSuiteBranchStore,
	commitDiscoveryStore core.CommitDiscoveryStore,
	logger lumber.Logger,
) core.DiscoveryStore {
	return &discoveryStore{
		db:                   db,
		testStore:            testStore,
		testSuiteStore:       testSuiteStore,
		testSuiteBranchStore: testSuiteBranchStore,
		testBranchStore:      testBranchStore,
		commitDiscoveryStore: commitDiscoveryStore,
		logger:               logger,
	}
}
func (ds *discoveryStore) Create(ctx context.Context,
	tests []*core.Test,
	testBranches []*core.TestBranch,
	suites []*core.TestSuite,
	suiteBranches []*core.TestSuiteBranch,
	commitDiscovery *core.CommitDiscovery) error {
	return ds.db.ExecuteTransactionWithRetry(ctx, maxRetries, delay, maxJitter, errMsg,
		func(tx *sqlx.Tx) error {
			if len(suites) > 0 {
				if err := ds.testSuiteStore.CreateInTx(ctx, tx, suites); err != nil {
					ds.logger.Errorf("failed to insert test suites, error: %v", err)
					return err
				}

				if err := ds.testSuiteBranchStore.CreateInTx(ctx, tx, suiteBranches); err != nil {
					ds.logger.Errorf("failed to insert suite and branch relation, error: %v", err)
					return err
				}
			}
			if len(tests) > 0 {
				if err := ds.testStore.CreateInTx(ctx, tx, tests); err != nil {
					ds.logger.Errorf("failed to insert tests, error: %v", err)
					return err
				}

				if err := ds.testBranchStore.CreateInTx(ctx, tx, testBranches); err != nil {
					ds.logger.Errorf("failed to insert test and branch relation, error: %v", err)
					return err
				}
			}
			if err := ds.commitDiscoveryStore.CreateInTx(ctx, tx, commitDiscovery); err != nil {
				ds.logger.Errorf("failed to insert entity in commit discovery store, error: %v", err)
				return err
			}
			return nil
		})
}
