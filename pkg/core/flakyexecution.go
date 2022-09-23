package core

import "context"

// FlakyExecutionStore is wrapper for executing all the queries related to flakyexecution in a transaction.
type FlakyExecutionStore interface {
	// Create persists the data in the datastore.
	Create(ctx context.Context,
		FlakyTestExecution []*FlakyTestExecution,
		blockTests []*BlockTest,
		buildID string) error
}
