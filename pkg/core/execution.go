package core

import "context"

// ExecutionStore is wrapper for executing all the queries related to execution in a transaction.
type ExecutionStore interface {
	// Create persists the data in the datastore.
	Create(ctx context.Context,
		testExecutions []*TestExecution,
		testSuiteExecutions []*TestSuiteExecution,
		buildID string) error
}
