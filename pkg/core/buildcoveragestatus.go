package core

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

// BuildCoverageStatus represents the coverage for a build.
type BuildCoverageStatus struct {
	BuildID             string    `db:"build_id"`
	BaseCommitID        string    `db:"base_commit_id"`
	CoverageAvailable   bool      `db:"coverage_available"`
	CoverageRecoverable bool      `db:"coverage_recoverable"`
	Created             time.Time `db:"created_at"`
	Updated             time.Time `db:"updated_at"`
}

// BuildCoverageStatusStore defines datastore operation for working with commit_coverage_status table
type BuildCoverageStatusStore interface {
	// Create persists the BuildCoverageStatus data into datastore.
	Create(ctx context.Context, ccStatus *BuildCoverageStatus) error
	// FindDependentCoverageJobs finds dependent coverage jobs for given commitIDs
	FindDependentCoverageJobsToRun(ctx context.Context, repoID string, commitIDs []string) ([]*CoverageInput, error)
	// FindPendingCoverageJobsToRun finds pending coverage jobs in the system to run (by orgID sorted)
	FindPendingCoverageJobsToRun(ctx context.Context) ([]*CoverageInput, error)
	// UpdateBulkCoverageAvailableTrueInTx sets coverage_available to true for all commitIDs for a given repoID
	UpdateBulkCoverageAvailableTrueInTx(ctx context.Context, tx *sqlx.Tx, repoID string, commitIDs []string) error
	// GetCoverageStatusCounts helps in assessing whether coverage data for a commit can be made available and hence do smart-selection
	GetCoverageStatusCounts(ctx context.Context, commitID, repoID string) (*int16, *int16, int, error)
}
