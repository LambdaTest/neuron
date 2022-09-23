package buildcoveragestatus

import (
	"context"
	"database/sql"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/jmoiron/sqlx"
)

type buildCoverageStatusStore struct {
	db     core.DB
	logger lumber.Logger
}

// New returns a new CommitCoverageStatusStore.
func New(db core.DB, logger lumber.Logger) core.BuildCoverageStatusStore {
	return &buildCoverageStatusStore{db: db, logger: logger}
}

func (bcs *buildCoverageStatusStore) Create(ctx context.Context, ccStatus *core.BuildCoverageStatus) error {
	return bcs.db.Execute(func(db *sqlx.DB) error {
		if _, err := db.NamedExecContext(ctx, insertQuery, ccStatus); err != nil {
			return err
		}
		return nil
	})
}

func (bcs *buildCoverageStatusStore) FindDependentCoverageJobsToRun(
	ctx context.Context,
	repoID string,
	commitIDs []string) ([]*core.CoverageInput, error) {
	coverageJobs := make([]*core.CoverageInput, 0)
	return coverageJobs, bcs.db.Execute(func(db *sqlx.DB) error {
		selectQuery, args, err := sqlx.In(`
			SELECT
				bcs.build_id,
				r.org_id,
				b.payload_address
			FROM
				build_coverage_status bcs
			JOIN
				build b ON bcs.build_id = b.id
			JOIN
				repositories r ON b.repo_id = r.id
			WHERE
				bcs.coverage_available = 0
				AND bcs.coverage_recoverable = 1
				AND r.id = ?
				AND bcs.base_commit_id IN (?)`, repoID, commitIDs)
		if err != nil {
			return err
		}
		rows, err := db.QueryxContext(ctx, selectQuery, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			coverageJob := new(core.CoverageInput)
			if err := rows.Scan(&coverageJob.BuildID, &coverageJob.OrgID, &coverageJob.PayloadAddress); err != nil {
				return err
			}
			coverageJobs = append(coverageJobs, coverageJob)
		}
		return nil
	})
}

func (bcs *buildCoverageStatusStore) FindPendingCoverageJobsToRun(ctx context.Context) ([]*core.CoverageInput, error) {
	coverageJobs := make([]*core.CoverageInput, 0)
	return coverageJobs, bcs.db.Execute(func(db *sqlx.DB) error {
		rows, err := db.QueryxContext(ctx, pendingCoverageJobs)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			coverageJob := new(core.CoverageInput)
			if err := rows.Scan(&coverageJob.BuildID, &coverageJob.OrgID, &coverageJob.PayloadAddress); err != nil {
				return err
			}
			coverageJobs = append(coverageJobs, coverageJob)
		}
		return nil
	})
}

func (bcs *buildCoverageStatusStore) UpdateBulkCoverageAvailableTrueInTx(
	ctx context.Context, tx *sqlx.Tx, repoID string, commitIDs []string) error {
	bulkUpdateQuery, args, err := sqlx.In(`
		UPDATE
			build_coverage_status bcs
		JOIN build b ON
			b.id = bcs.build_id
		JOIN repositories r ON
			r.id = b.repo_id
		SET
			bcs.coverage_available = 1,
			bcs.coverage_recoverable = 1
		WHERE
			bcs.coverage_available = 0
			AND r.id = ?
			AND b.commit_id IN (?)`, repoID, commitIDs)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, bulkUpdateQuery, args...); err != nil {
		return err
	}
	return nil
}

func (bcs *buildCoverageStatusStore) GetCoverageStatusCounts(
	ctx context.Context,
	commitID, repoID string) (coverageAvailable, coverageRecoverable *int16, taskCount int, err error) {
	return coverageAvailable, coverageRecoverable, taskCount, bcs.db.Execute(func(db *sqlx.DB) error {
		var dbCoverageAvailable, dbCoverageRecoverable sql.NullInt16
		dbTaskCount := new(int)
		row := db.QueryRowxContext(ctx, coverageStatusCountsQuery, commitID, repoID)
		if err := row.Scan(&dbCoverageAvailable, &dbCoverageRecoverable, dbTaskCount); err != nil {
			return err
		}
		if dbCoverageAvailable.Valid {
			coverageAvailable = &dbCoverageAvailable.Int16
		}
		if dbCoverageRecoverable.Valid {
			coverageRecoverable = &dbCoverageRecoverable.Int16
		}
		taskCount = *dbTaskCount
		return nil
	})
}

const insertQuery = `
INSERT
	INTO
	build_coverage_status (
		build_id,
		base_commit_id,
		coverage_recoverable,
		created_at,
		updated_at
	)
VALUES (
	:build_id,
	:base_commit_id,
	:coverage_recoverable,
	:created_at,
	:updated_at
) ON
DUPLICATE KEY
UPDATE
	coverage_recoverable =
	IF(
		coverage_recoverable = 1,
		coverage_recoverable,
	VALUES(coverage_recoverable)
	),
	updated_at =
	IF(
		coverage_recoverable = 1,
		updated_at,
	VALUES(updated_at)
	)
`

const coverageStatusCountsQuery = `
SELECT
	sum(bcs.coverage_available),
	sum(bcs.coverage_recoverable),
	count(b.id)
FROM
	build_coverage_status bcs
RIGHT JOIN build b ON
	bcs.build_id = b.id
JOIN repositories r ON
	b.repo_id = r.id
WHERE
	b.commit_id = ?
	AND r.id = ?
`

const pendingCoverageJobs = `
SELECT
	bcs.build_id,
	r.org_id,
	b.payload_address
FROM
	build_coverage_status bcs
JOIN
	build b ON bcs.build_id = b.id
JOIN
	repositories r ON b.repo_id = r.id
JOIN
	test_coverage tc ON r.id = tc.repo_id AND bcs.base_commit_id = tc.commit_id
WHERE
	bcs.coverage_available = 0
	AND bcs.coverage_recoverable = 1
ORDER BY r.org_id
`
