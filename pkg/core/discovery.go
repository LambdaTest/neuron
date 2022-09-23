package core

import "context"

// DiscoveryStore is wrapper for executing all the queries related to test discovery in a transaction.
type DiscoveryStore interface {
	// Create persists the data in the datastore.
	Create(ctx context.Context,
		tests []*Test,
		testBranches []*TestBranch,
		suites []*TestSuite,
		suiteBranches []*TestSuiteBranch,
		commitDiscovery *CommitDiscovery) error
}
