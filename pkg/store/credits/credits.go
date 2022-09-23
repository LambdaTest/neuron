package credits

import (
	"context"
	"math"
	"time"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/jmoiron/sqlx"
)

const timeOffset = 0.499
const argLength = 3

type creditsUsageStore struct {
	db     core.DB
	logger lumber.Logger
}

// New returns a new creditsUsageStore
func New(db core.DB, logger lumber.Logger) core.CreditsUsageStore {
	return &creditsUsageStore{db: db, logger: logger}
}

func (c *creditsUsageStore) CreateInTx(ctx context.Context, tx *sqlx.Tx, usage *core.CreditsUsage) error {
	if _, err := tx.NamedExecContext(ctx, insertQuery, usage); err != nil {
		return errs.SQLError(err)
	}
	return nil
}

func (c *creditsUsageStore) FindByOrgOrUser(ctx context.Context,
	user, orgID string,
	startDate, endDate time.Time,
	offset, limit int) (usages []*core.CreditsUsage, err error) {
	usages = make([]*core.CreditsUsage, 0, limit)
	return usages, c.db.Execute(func(db *sqlx.DB) error {
		args := make([]interface{}, 0, argLength)
		var query string
		if user == "" {
			args = append(args, orgID)
			query = selectByOrgQuery
		} else {
			args = append(args, user)
			query = selectByUserQuery
		}
		args = append(args, startDate, endDate, limit, offset)
		rows, err := db.QueryxContext(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			usage := new(core.CreditsUsage)
			err := rows.Scan(&usage.CommitID,
				&usage.OrgID,
				&usage.User,
				&usage.Tier, &usage.Duration, &usage.Created, &usage.BuildID, &usage.RepoName, &usage.BuildTag)
			if err != nil {
				return errs.SQLError(err)
			}
			if rows.Err() != nil {
				return rows.Err()
			}
			creditsPM := core.TaskTierConfigMapping[usage.Tier].CreditsPM
			usage.Consumed = creditsPM * int32(math.Round(usage.Duration/60+timeOffset))
			usage.MachineConfig = core.TaskTierConfigMapping[usage.Tier].String()
			usages = append(usages, usage)
		}
		return nil
	})
}

func (c *creditsUsageStore) FindTotalConsumedCredits(ctx context.Context, orgID string,
	startDate, endDate time.Time) (totalConsumedCredits int32, err error) {
	return totalConsumedCredits, c.db.Execute(func(db *sqlx.DB) error {
		args := make([]interface{}, 0, argLength)
		args = append(args, orgID, startDate, endDate)
		rows, err := db.QueryxContext(ctx, selectFindTotalConsumedQuery, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var duration float64
			var tier core.Tier
			err := rows.Scan(&tier, &duration)
			if err != nil {
				return errs.SQLError(err)
			}
			if rows.Err() != nil {
				return rows.Err()
			}
			creditsPM := core.TaskTierConfigMapping[tier].CreditsPM
			totalConsumedCredits += creditsPM * int32(math.Round(duration/60+timeOffset))
		}
		return nil
	})
}

const insertQuery = `
INSERT
	INTO
	credits_usage (
		id,
		org_id,
		task_id,
		created_at,
		updated_at,
		user
	)
SELECT
	:id,
	:org_id,
	:task_id,
	:created_at,
	:updated_at,
	c.author_name
FROM
	task t
JOIN build b ON
	b.id = t.build_id
JOIN git_commits c ON
	c.commit_id = b.commit_id
	AND c.repo_id = b.repo_id
WHERE
	t.id = :task_id
`

const selectByUserQuery = `
SELECT
	b.commit_id,
	r.org_id,
	any_value(c.user),
	b.tier,
	COALESCE(SUM(TIME_TO_SEC(TIMEDIFF(t.end_time, t.start_time))), 0) duration,
	b.created_at,
	b.id,
	r.name,
	b.tag
FROM
	credits_usage c
JOIN task t ON
	c.task_id = t.id
JOIN build b ON
	b.id = t.build_id
JOIN repositories r ON
	r.id = b.repo_id
WHERE
	user = ?
	AND c.created_at >= ?
	AND c.created_at <= ?
	AND b.status NOT IN ('initiating', 'running')
GROUP BY
	b.id
ORDER BY
	b.created_at DESC
LIMIT ? OFFSET ?
`

const selectByOrgQuery = `
SELECT
	b.commit_id,
	r.org_id,
	any_value(c.user),
	b.tier,
	COALESCE(SUM(TIME_TO_SEC(TIMEDIFF(t.end_time, t.start_time))), 0) duration,
	b.created_at,
	b.id,
	r.name,
	b.tag
FROM
	credits_usage c
JOIN task t ON
	c.task_id = t.id
JOIN build b ON
	b.id = t.build_id
JOIN repositories r ON
	r.id = b.repo_id
WHERE
	c.org_id = ?
	AND c.created_at >= ?
	AND c.created_at <= ?
	AND b.status NOT IN ('initiating', 'running')
GROUP BY
	b.id
ORDER BY
	b.created_at DESC
LIMIT ? OFFSET ?
`

const selectFindTotalConsumedQuery = `SELECT
	b.tier,
	COALESCE(SUM(TIME_TO_SEC(TIMEDIFF(t.end_time, t.start_time))), 0) duration
FROM
	credits_usage c
JOIN organizations o ON
	c.org_id = o.id
JOIN task t ON
	c.task_id = t.id
JOIN build b ON
	b.id = t.build_id
WHERE
	o.id = ?
	AND c.created_at >= ?
	AND c.created_at <= ?
	AND b.status NOT IN ('initiating', 'running')
GROUP BY
	b.id
`
