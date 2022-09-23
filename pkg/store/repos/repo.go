package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/LambdaTest/neuron/pkg/constants"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/jmoiron/sqlx"
)

type repoStore struct {
	db     core.DB
	logger lumber.Logger
}

// New returns a new repoStore
func New(db core.DB, logger lumber.Logger) core.RepoStore {
	return &repoStore{db: db, logger: logger}
}

func (r *repoStore) Create(ctx context.Context, repo *core.Repository) error {
	return r.db.Execute(func(db *sqlx.DB) error {
		r.logger.Debugf("repo %+v", repo)
		if _, err := db.NamedExecContext(ctx, insertQuery, repo); err != nil {
			return errs.SQLError(err)
		}
		return nil
	})
}

func (r *repoStore) Find(ctx context.Context, orgID, name string) (*core.Repository, error) {
	repo := &core.Repository{}
	return repo, r.db.Execute(func(db *sqlx.DB) error {
		query := selectQuery
		rows := db.QueryRowxContext(ctx, query, orgID, name)
		return rows.StructScan(repo)
	})
}

func (r *repoStore) FindByID(ctx context.Context, repoID string) (*core.Repository, error) {
	repo := &core.Repository{}
	return repo, r.db.Execute(func(db *sqlx.DB) error {
		rows := db.QueryRowxContext(ctx, selectByIDQuery, repoID)
		return rows.StructScan(repo)
	})
}

func (r *repoStore) FindIfActive(ctx context.Context, orgID, name string) (*core.Repository, error) {
	repo := &core.Repository{}
	return repo, r.db.Execute(func(db *sqlx.DB) error {
		selectQuery := "SELECT id, org_id, name, webhook_secret, link, active, mask FROM repositories WHERE org_id=? AND name=? AND active=1"
		rows := db.QueryRowxContext(ctx, selectQuery, orgID, name)
		return rows.StructScan(repo)
	})
}
func (r *repoStore) FindAllActiveMap(ctx context.Context, orgID string) (map[string]struct{}, error) {
	repo := make(map[string]struct{})
	return repo, r.db.Execute(func(db *sqlx.DB) error {
		rows, err := db.QueryxContext(ctx, selectActiveQuery, orgID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var name string
			if err = rows.Scan(&name); err != nil {
				r.logger.Errorf("error in scanning rows while finding active repos, error: %v", err)
				return err
			}
			repo[name] = struct{}{}
		}
		return nil
	})
}

func (r *repoStore) FindAllActive(ctx context.Context, orgID, searchText string, offset, limit int) ([]*core.Repository, error) {
	repos := make([]*core.Repository, 0, limit)
	return repos, r.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"org":    orgID,
			"offset": offset,
			"limit":  limit,
			"repo":   "%" + searchText + "%",
		}
		query := selectActiveByOrgQuery
		if searchText != "" {
			query += " AND r.name LIKE :repo "
		}
		query += " GROUP BY r.id ORDER BY created DESC LIMIT :limit OFFSET :offset"
		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var jsonRaw string
			repo := new(core.Repository)
			repo.Meta = new(core.RepoMeta)
			repo.Meta.BuildMeta = new(core.BuildMeta)
			if err = rows.Scan(
				&repo.ID,
				&repo.Name,
				&repo.Active,
				&repo.Link,
				&repo.Meta.Total,
				&repo.Meta.TotalTests,
				&repo.Meta.Error,
				&repo.Meta.Failed,
				&repo.Meta.Passed,
				&repo.Meta.Initiating,
				&repo.Meta.Running,
				&repo.Meta.Aborted,
				&repo.Meta.AvgDuration,
				&jsonRaw,
				&repo.Updated,
				&repo.Created); err != nil {
				return errs.SQLError(err)
			}
			freqCount := make([]int, constants.MonthInterval)
			var relativeBuildsArray []int
			if err := json.Unmarshal([]byte(jsonRaw), &relativeBuildsArray); err != nil {
				r.logger.Errorf("failed to unmarshall json into days array orgID %s, error %v", orgID, err)
				return err
			}

			for _, value := range relativeBuildsArray {
				if value <= constants.MonthInterval && value > 0 {
					freqCount[value-1]++
				}
			}
			if rows.Err() != nil {
				return rows.Err()
			}
			repo.RepoGraph = freqCount
			repos = append(repos, repo)
		}
		return nil
	})
}

func (r *repoStore) FindByCommitID(ctx context.Context, commitID string) (repoName string, authorName string, err error) {
	return repoName, authorName, r.db.Execute(func(db *sqlx.DB) error {
		query := "SELECT r.name,g.author_name from repositories r join git_commits g on r.id=g.repo_id where g.commit_id=?"
		rows := db.QueryRowContext(ctx, query, commitID)

		if err := rows.Scan(&repoName, &authorName); err != nil {
			return err
		}
		return nil
	})
}

func (r *repoStore) FindActiveByName(ctx context.Context,
	repoName,
	orgName string,
	gitProvider core.SCMDriver) (repo *core.Repository, err error) {
	repo = new(core.Repository)
	return repo, r.db.Execute(func(db *sqlx.DB) error {
		rows := db.QueryRowxContext(ctx, selectActiveByOrgNameQuery, repoName, orgName, gitProvider)
		if err := rows.StructScan(repo); err != nil {
			return errs.SQLError(err)
		}
		return nil
	})
}

func (r *repoStore) FindByBuildID(ctx context.Context, buildID string) (*core.Repository, error) {
	repo := &core.Repository{}
	return repo, r.db.Execute(func(db *sqlx.DB) error {
		selectQuery := `SELECT r.id,
       					r.org_id,
       					r.admin_id,
       					r.mask
						FROM repositories r
						JOIN build b ON r.id = b.repo_id
						WHERE b.id = ?`
		rows := db.QueryRowxContext(ctx, selectQuery, buildID)
		return rows.StructScan(repo)
	})
}

func (r *repoStore) UpdateStatus(ctx context.Context, repoConfig core.RepoSettingsInfo, orgID, userID string) error {
	return r.db.Execute(func(db *sqlx.DB) error {
		args := []interface{}{repoConfig.Strict, repoConfig.ConfigFileName, repoConfig.JobView,
			repoConfig.RepoName, orgID, userID}
		if _, err := db.ExecContext(ctx, updateConfigInfo, args...); err != nil {
			return errs.SQLError(err)
		}
		return nil
	})
}

func (r *repoStore) FindBadgeData(ctx context.Context, repoName, orgName, branchName, buildID,
	gitProvider string) (*core.Badge, error) {
	badge := &core.Badge{}
	return badge, r.db.Execute(func(db *sqlx.DB) error {
		args := []interface{}{repoName, orgName, gitProvider}
		query := findBadgeDataQuery
		buildWhere := ""
		branchWhere := ""
		if buildID != "" {
			buildWhere = " AND b.id = ? "
			args = append(args, buildID)
		}
		if branchName != "" {
			branchWhere = " AND b.branch_name = ? "
			args = append(args, branchName)
		}
		query = fmt.Sprintf(query, buildWhere, branchWhere)

		rows := db.QueryRowxContext(ctx, query, args...)
		var tests, duration, testDuration int
		if err := rows.Scan(&tests,
			&badge.BuildStatus,
			&duration,
			&testDuration,
			&badge.Passed,
			&badge.Failed,
			&badge.Skipped,
			&badge.BuildID); err != nil {
			return err
		}
		badge.TotalTests = tests
		badge.ExecutionTime = testDuration
		if buildID != "" {
			durMillisecond := time.Millisecond * time.Duration(testDuration)
			badge.Duration = durMillisecond.String()
		} else {
			durSecond := time.Second * time.Duration(duration)
			badge.Duration = durSecond.String()
		}
		return nil
	})
}

func (r *repoStore) FindBadgeDataFlaky(ctx context.Context, repoName, orgName,
	branchName, gitProvider string) (*core.BadgeFlaky, error) {
	badge := &core.BadgeFlaky{}
	return badge, r.db.Execute(func(db *sqlx.DB) error {
		args := []interface{}{repoName, orgName, gitProvider}
		query := findBadgeDataFlakyQuery
		branchWhere := ""
		if branchName != "" {
			branchWhere = " AND b.branch_name = ? "
			args = append(args, branchName)
		}
		query = fmt.Sprintf(query, branchWhere)

		rows := db.QueryRowxContext(ctx, query, args...)
		if err := rows.Scan(&badge.BuildStatus,
			&badge.FlakyTests,
			&badge.NonFlakyTests,
			&badge.Quarantined,
			&badge.BuildID); err != nil {
			return err
		}
		return nil
	})
}

func (r *repoStore) RemoveRepo(ctx context.Context, repoName, orgID, userID string) error {
	return r.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo":    repoName,
			"org_id":  orgID,
			"user_id": userID,
		}
		query := updateRepoSettingsQuery
		result, err := db.NamedExecContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}
		if rowCount, err := result.RowsAffected(); err != nil {
			return errs.SQLError(err)
		} else if rowCount == 0 {
			return errs.ErrNotFound
		}
		return nil
	})
}

func (r *repoStore) FindTokenPathInfo(ctx context.Context, repoID string) (*core.TokenPathInfo, error) {
	tokenPathInfo := &core.TokenPathInfo{}
	return tokenPathInfo, r.db.Execute(func(db *sqlx.DB) error {
		rows := db.QueryRowxContext(ctx, selectTokenPathInfoQuery, repoID)
		return rows.StructScan(tokenPathInfo)
	})
}

const insertQuery = `INSERT
INTO
repositories(id,
org_id,
name,
link,
tas_file_name,
webhook_secret,
active,
admin_id,
mask,
private,
job_view,
created_at,
updated_at)
VALUES (:id,
:org_id,
:name,
:link,
:tas_file_name,
:webhook_secret,
:active,
:admin_id,
:mask,
:private,
:job_view,
:created_at,
:updated_at) ON
DUPLICATE KEY
UPDATE
updated_at =:updated_at,
webhook_secret =:webhook_secret,
tas_file_name =:tas_file_name,
active =:active,
mask =:mask,
private=:private`

const selectActiveQuery = `SELECT
r.name
FROM
repositories r
WHERE
r.org_id = ?
AND r.active = 1`

const selectActiveByOrgQuery = `
WITH test_meta AS 
(
SELECT
	r.id,
	COUNT(DISTINCT t.id) total_tests
FROM
		repositories r
JOIN test t ON
		t.repo_id = r.id
WHERE
		r.org_id = :org
GROUP BY
	r.id 
),
build_meta AS (
SELECT
	r.id,
	COUNT(DISTINCT b.id) total_builds,
	COUNT(DISTINCT CASE WHEN b.status = 'error' THEN b.id END) error,
	COUNT(DISTINCT CASE WHEN b.status = 'failed' THEN b.id END) failed,
	COUNT(DISTINCT CASE WHEN b.status = 'passed' THEN b.id END) passed,
	COUNT(DISTINCT CASE WHEN b.status = 'initiating' THEN b.id END) initiating,
	COUNT(DISTINCT CASE WHEN b.status = 'running' THEN b.id END) running,
	COUNT(DISTINCT CASE WHEN b.status = 'aborted' THEN b.id END) aborted,
	IFNULL(AVG(TIME_TO_SEC(TIMEDIFF(b.end_time, b.start_time))), 0) average_time,
	IFNULL(MAX(b.created_at), TIMESTAMP("0001-01-01")) created
FROM
		repositories r
JOIN build b ON
		b.repo_id = r.id
WHERE
		r.org_id = :org
GROUP BY
	r.id
),
repo_graphs AS (
SELECT
	r.id,
	JSON_ARRAYAGG(DATEDIFF(b.created_at, NOW() + INTERVAL -1 MONTH)) graph,
	IFNULL(MAX(b.updated_at), TIMESTAMP("0001-01-01")) updated_at
FROM
	repositories r
JOIN build b ON
	b.repo_id = r.id
WHERE
	b.created_at >= NOW() + INTERVAL -1 MONTH
	AND
	r.org_id = :org
	AND
	r.active = 1
GROUP BY
	r.id
)
SELECT
	r.id,
	r.name,
	r.active,
	r.link,
	COALESCE(b.total_builds, 0) total_builds,
	COALESCE(t.total_tests, 0) total_tests,
	COALESCE(b.error, 0) error,
	COALESCE(b.failed, 0) failed,
	COALESCE(b.passed, 0) passed,
	COALESCE(b.initiating, 0) initiating,
	COALESCE(b.running, 0) running,
	COALESCE(b.aborted, 0) aborted,
	COALESCE(b.average_time, 0) average_time,
	IFNULL(repo_graphs.graph, JSON_ARRAY()),
	IFNULL(repo_graphs.updated_at, TIMESTAMP("0001-01-01")),
	COALESCE(b.created, TIMESTAMP("0001-01-01")) created
FROM
	repositories r
LEFT JOIN test_meta t ON
	r.id = t.id
LEFT JOIN build_meta b ON
	b.id = r.id
LEFT JOIN repo_graphs ON
	r.id = repo_graphs.id
WHERE
	r.org_id = :org
	AND r.active = 1
`

const updateConfigInfo = `
UPDATE
	repositories r
SET
	r.strict = ?,
	r.tas_file_name = ?,
	r.job_view = ?,
	r.updated_at = NOW()
WHERE
	r.name = ?
	AND r.org_id = ?
	AND r.admin_id = ?
`
const selectQuery = `
SELECT
	id,
	org_id,
	name,
	webhook_secret,
	link,
	active,
	strict,
	tas_file_name,
	admin_id,
	job_view
FROM
	repositories
WHERE
	org_id =?
	AND name =?
`

const selectByIDQuery = `
SELECT
	id,
	org_id,
	strict,
	name,
	webhook_secret,
	link,
	active,
	private,
	tas_file_name,
	mask,
	admin_id,
	collect_coverage
FROM
	repositories
WHERE
	id = ?
	AND active = 1
`

const selectActiveByOrgNameQuery = `
SELECT
	r.id,
	r.org_id,
	r.private,
	r.name,
	r.link,
	r.tas_file_name
FROM
	repositories r
JOIN organizations o ON
	o.id = r.org_id
WHERE
	r.active = 1
	AND r.name = ?
	AND o.name = ?
	AND o.git_provider = ?`

const findBadgeDataQuery = `
WITH latest_build AS (
	SELECT
		b.id,
		b.status,
		b.start_time,
		b.end_time 
	FROM
		build b
	JOIN repositories r ON
		r.id = b.repo_id
	JOIN organizations o ON
		o.id = r.org_id
	WHERE
		r.name = ?
		AND o.name = ?
		AND o.git_provider = ?
		%s
		%s
	ORDER BY
		b.updated_at DESC
	LIMIT 1
	)
	SELECT 
		COALESCE(COUNT(te.test_id), 0) total_tests_impacted,
		lb.status status,
		COALESCE(TIME_TO_SEC(TIMEDIFF(lb.end_time, lb.start_time)), 0) duration,
		COALESCE(SUM(te.duration), 0) testduration,
		COALESCE(COUNT(DISTINCT CASE WHEN te.status = 'passed' THEN te.test_id END),0) passed,
		COALESCE(COUNT(DISTINCT CASE WHEN te.status = 'failed' THEN te.test_id END),0) failed,
		COALESCE(COUNT(DISTINCT CASE WHEN te.status = 'skipped' THEN te.test_id END),0) skipped,
		lb.id
	FROM
		latest_build lb
	LEFT JOIN test_execution te ON
		lb.id = te.build_id
	GROUP BY lb.id `

const updateRepoSettingsQuery = `
UPDATE
	repositories r
SET
	r.active = 0,
	r.updated_at = NOW()
WHERE
	r.name = :repo
	AND r.org_id = :org_id 
	AND r.admin_id = :user_id `

// nolint:gosec
const selectTokenPathInfoQuery = `
SELECT
	r.org_id,
	o.name as org_name,
	r.admin_id,
	gu.mask as user_mask,
	gu.git_provider
FROM
	repositories r
LEFT JOIN git_users gu ON
	gu.id = r.admin_id
LEFT JOIN organizations o ON
	o.id = r.org_id
WHERE
	r.id = ? ;`

const findBadgeDataFlakyQuery = `
WITH latest_build AS (
	SELECT
		b.id,
		b.status,
		b.start_time,
		b.end_time 
	FROM
		build b
	JOIN repositories r ON
		r.id = b.repo_id
	JOIN organizations o ON
		o.id = r.org_id
	WHERE
		r.name = ?
		AND o.name = ?
		AND o.git_provider = ?
		%s
	ORDER BY
		b.updated_at DESC
	LIMIT 1
	)
	SELECT 
		lb.status status,
		COALESCE(COUNT(DISTINCT CASE WHEN fte.status = 'flaky' THEN fte.test_id END),0) flaky,
		COALESCE(COUNT(DISTINCT CASE WHEN fte.status = 'nonflaky' THEN fte.test_id END),0) nonflaky,
		COALESCE(COUNT(DISTINCT CASE WHEN fte.status = 'quarantined' THEN fte.test_id END),0) quarantined,
		lb.id
	FROM
		latest_build lb
	JOIN flaky_test_execution fte ON
		lb.id = fte.build_id
	GROUP BY lb.id 	
`
