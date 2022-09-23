package core

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"
)

// TestCoverage represents the code coverage for each testfile.
type TestCoverage struct {
	ID            string          `db:"id"`
	CommitID      string          `db:"commit_id"`
	RepoID        string          `db:"repo_id"`
	Blob          string          `db:"blob_link"`
	TotalCoverage json.RawMessage `db:"total_coverage"`
	Created       time.Time       `db:"created_at"`
	Updated       time.Time       `db:"updated_at"`
}

// CoverageInput represents the input for the repoqueue
type CoverageInput struct {
	BuildID        string
	OrgID          string
	PayloadAddress string
}

// TestCoverageStore defines datastore operation for working with test coverage
type TestCoverageStore interface {
	// CreateInTx persists the coverage data into datastore.
	CreateInTx(ctx context.Context, tx *sqlx.Tx, coverage []*TestCoverage) error
	// FindCommitCoverage finds the coverage data into datastore by commitID.
	FindCommitCoverage(ctx context.Context, repoID, commitID string) (*TestCoverage, error)
}

// CoverageManager acts as a layer between the coverage APIs and coverage store that contains non-trivial business logic
type CoverageManager interface {
	// IsBaseCommitCoverageAvailable helps in determine whether coverageData is available or can be made available later
	IsBaseCommitCoverageAvailable(ctx context.Context, baseCommit string, repo *Repository) (bool, error)
	// CanRunCoverageImmediately checks if coverage job can be run immediately
	CanRunCoverageImmediately(ctx context.Context, baseCommit, repoID string) (bool, error)
	// RunCoverageJob runs the coverage job for a build
	RunCoverageJob(ctx context.Context, input *CoverageInput) error
	// InsertCoverageData performs post coverage-collection steps
	InsertCoverageData(ctx context.Context, data []*TestCoverage) error
	// RunPendingCoverageJobs finds and runs all the pending coverage jobs (recoverable jobs whose base commit's build coverage is recorded)
	RunPendingCoverageJobs(ctx context.Context) error
}
