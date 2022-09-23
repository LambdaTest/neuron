package builds

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"gopkg.in/guregu/null.v4/zero"
)

const (
	maxRetries = 3
	delay      = 250 * time.Millisecond
	maxJitter  = 100 * time.Millisecond
	errMsg     = "failed to perform build transaction"
)

type buildStore struct {
	db      core.DB
	logger  lumber.Logger
	runner  core.K8sRunner
	redisDB core.RedisDB
}

// New returns a new BuildStore
func New(db core.DB, redisDB core.RedisDB, runner core.K8sRunner, logger lumber.Logger) core.BuildStore {
	return &buildStore{db: db, redisDB: redisDB, runner: runner, logger: logger}
}

func (b *buildStore) CreateInTx(ctx context.Context, tx *sqlx.Tx, build *core.Build) error {
	if _, err := tx.NamedExecContext(ctx, insertQuery, build); err != nil {
		return err
	}
	return nil
}

func (b *buildStore) FindByBuildID(ctx context.Context, buildID string) (*core.Build, error) {
	build := new(core.Build)
	return build, b.db.Execute(func(db *sqlx.DB) error {
		rows := db.QueryRowxContext(ctx, findByBuildIDQuery, buildID)
		if err := rows.StructScan(build); err != nil {
			return errs.SQLError(err)
		}
		return nil
	})
}

func (b *buildStore) CheckIfPassedBuildExistsInTx(ctx context.Context, tx *sqlx.Tx, repoID string) (bool, error) {
	var exists bool
	row := tx.QueryRowxContext(ctx, checkIfPassedBuildExists, repoID)
	err := row.Scan(&exists)
	if err != nil && !errors.Is(err, errs.ErrRowsNotFound) {
		return false, err
	}
	return exists, nil
}

func (b *buildStore) Update(ctx context.Context, build *core.Build) error {
	return b.db.Execute(func(db *sqlx.DB) error {
		if _, err := db.NamedExecContext(ctx, updateQuery, build); err != nil {
			return err
		}
		return nil
	})
}

func (b *buildStore) MarkStarted(ctx context.Context, build *core.Build) error {
	return b.db.Execute(func(db *sqlx.DB) error {
		markStartedQuery := updateQuery + ` AND start_time IS NULL AND status='initiating'`
		if _, err := db.NamedExecContext(ctx, markStartedQuery, build); err != nil {
			return err
		}
		return nil
	})
}

func (b *buildStore) MarkStopped(ctx context.Context, build *core.Build, orgID string) error {
	return b.db.ExecuteTransactionWithRetry(ctx, maxRetries, delay, maxJitter, errMsg, func(tx *sqlx.Tx) error {
		return b.MarkStoppedInTx(ctx, tx, build, orgID)
	})
}

func (b *buildStore) MarkStoppedInTx(ctx context.Context, tx *sqlx.Tx, build *core.Build, orgID string) error {
	if _, err := tx.NamedExecContext(ctx, markStoppedQuery, build); err != nil {
		return err
	}
	b.deleteCacheAndPVC(ctx, build, orgID)
	return nil
}

func (b *buildStore) UpdateTier(ctx context.Context, build *core.Build) error {
	return b.db.ExecuteTransactionWithRetry(ctx, maxRetries, delay, maxJitter, errMsg, func(tx *sqlx.Tx) error {
		if _, err := tx.NamedExecContext(ctx, updateTierQuery, build); err != nil {
			return err
		}
		key := utils.GetBuildHashKey(build.ID)
		b.redisDB.Client().HSet(ctx, key, map[string]interface{}{"tier": string(build.Tier)})
		return nil
	})
}

func (b *buildStore) deleteCacheAndPVC(ctx context.Context, build *core.Build, orgID string) {
	buildCache, err := b.GetBuildCache(ctx, build.ID)
	if err != nil {
		b.logger.Errorf("failed to find build details in redis for orgID %s, buildID %s, error: %v",
			orgID, build.ID, err)
	}
	if buildCache != nil {
		// delete build cache from redis
		if err := b.DeleteBuildCache(ctx, build.ID); err != nil {
			b.logger.Errorf("failed to delete build cache for buildID %s, orgID %s, error %v",
				build.ID, orgID, err)
		}
	}
	// incase of build error we delete the pvc as well, as coverage will not run.
	if build.Status == core.BuildError || build.Status == core.BuildAborted || (buildCache != nil && !buildCache.CollectCoverage) {
		runnerOptions := &core.RunnerOptions{
			NameSpace:                 utils.GetRunnerNamespaceFromOrgID(orgID),
			PersistentVolumeClaimName: utils.GetBuildHashKey(build.ID),
			Vault: &core.VaultOpts{
				TokenSecretName: utils.GetTokenSecretName(build.ID),
			}}
		if buildCache != nil && buildCache.SecretPath != "" {
			runnerOptions.Vault.RepoSecretName = utils.GetRepoSecretName(build.ID)
		}

		if err := b.runner.Cleanup(ctx, runnerOptions); err != nil {
			b.logger.Errorf("failed to perform cleanup of k8s resources for buildID %s, orgID %s, error %v",
				build.ID, orgID, err)
		}
	}
}

func (b *buildStore) StopStaleBuildsInTx(ctx context.Context, tx *sqlx.Tx, timeout time.Duration) (int64, error) {
	var updated int64
	// Need to do this because of https://github.com/go-sql-driver/mysql/issues/1217
	query := fmt.Sprintf(markStaleStoppedQuery, timeout.Hours())
	res, err := tx.ExecContext(ctx, query, errs.GenericJobTimeoutErrRemark.Error())
	if err != nil {
		return 0, err
	}
	updated, err = res.RowsAffected()
	return updated, err
}

func (b *buildStore) FindByRepo(ctx context.Context, repoName, orgID, buildID, branchName, statusFilter, searchID string,
	authorsNames, tags []string, lastSeenTime time.Time, limit int) ([]*core.Build, error) {
	builds := make([]*core.Build, 0)
	return builds, b.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo":           repoName,
			"org_id":         orgID,
			"branch":         branchName,
			"build_id":       buildID,
			"build_status":   statusFilter,
			"search_text":    "%" + searchID + "%",
			"limit":          limit,
			"last_seen_time": lastSeenTime,
		}
		query := findByRepoQuery
		branchWhere, buildIDWhere, buildStatusWhere, searchIDWhere := "", "", "", ""
		authorWhere, tagsWhere, createdAtWhere := "", "", ""
		if branchName != "" {
			branchWhere = branchCondition
		}
		if searchID != "" {
			searchIDWhere = ` AND ( b.commit_id LIKE :search_text OR b.id LIKE :search_text )`
		}
		if statusFilter != "" {
			buildStatusWhere = ` AND b.status = :build_status`
		}
		if buildID != "" {
			buildIDWhere = ` AND b.id = :build_id`
		}
		if len(tags) > 0 {
			tagsWhere = ` AND build.tag IN (%s)`
			inClause := ""
			inClause, args = utils.AddMultipleFilters("buildTags", tags, args)
			tagsWhere = fmt.Sprintf(tagsWhere, inClause)
		}
		if len(authorsNames) > 0 {
			authorWhere = ` AND build.author_name IN (%s)`
			inClause := ""
			inClause, args = utils.AddMultipleFilters("authorsNames", authorsNames, args)
			authorWhere = fmt.Sprintf(authorWhere, inClause)
		}
		if !lastSeenTime.IsZero() {
			createdAtWhere = ` AND build.created_at <= :last_seen_time`
		}
		query = fmt.Sprintf(query, branchWhere, searchIDWhere, buildStatusWhere, buildIDWhere, tagsWhere, authorWhere, createdAtWhere)
		rows, err := db.NamedQueryContext(ctx, query, args)

		if err != nil {
			return errs.SQLError(err)
		}

		defer rows.Close()
		for rows.Next() {
			b := new(core.Build)
			b.Meta = new(core.ExecutionMeta)
			b.FlakyMeta = new(core.FlakyExecutionMetadata)
			if err = rows.Scan(&b.BuildNum,
				&b.ID, &b.CommitID, &b.CommitAuthor,
				&b.CommitMessage, &b.Updated, &b.Created,
				&b.StartTime, &b.EndTime, &b.Status,
				&b.Remark, &b.JobView, &b.Meta.Total, &b.Meta.TotalTests,
				&b.Meta.TotalTestDuration, &b.Meta.Passed,
				&b.Meta.Failed, &b.Meta.Skipped,
				&b.Meta.Blocklisted, &b.Meta.Quarantined,
				&b.Meta.Aborted,
				&b.Tag,
				&b.Branch,
				&b.FlakyMeta.ImpactedTests,
				&b.FlakyMeta.FlakyTests,
				&b.FlakyMeta.Skipped,
				&b.FlakyMeta.Blocklisted,
				&b.FlakyMeta.Aborted); err != nil {
				return errs.SQLError(err)
			}
			b.Meta.Unimpacted = b.Meta.TotalTests - b.Meta.Total
			if b.FlakyMeta.ImpactedTests == 0 {
				b.FlakyMeta.OverAllFlakiness = 0
			} else {
				b.FlakyMeta.OverAllFlakiness = float32(b.FlakyMeta.FlakyTests) / float32(b.FlakyMeta.ImpactedTests)
			}
			builds = append(builds, b)
		}

		if len(builds) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
}

func (b *buildStore) FindByCommit(ctx context.Context,
	commitID,
	repoName,
	orgID,
	branchName string, lastSeenTime time.Time, limit int) ([]*core.Build, error) {
	builds := make([]*core.Build, 0)
	return builds, b.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo":           repoName,
			"org_id":         orgID,
			"branch":         branchName,
			"commit_id":      commitID,
			"limit":          limit,
			"last_seen_time": lastSeenTime,
		}
		query := findByCommitQuery
		branchWhere := ""
		lastSeenWhere := ""
		if branchName != "" {
			branchWhere = branchCondition
		}
		if !lastSeenTime.IsZero() {
			lastSeenWhere = createdAtCondition
		}
		query = fmt.Sprintf(query, branchWhere, lastSeenWhere)
		rows, err := db.NamedQueryContext(ctx, query, args)

		if err != nil {
			return errs.SQLError(err)
		}

		defer rows.Close()
		for rows.Next() {
			b := new(core.Build)
			if err = rows.Scan(&b.BuildNum, &b.ID, &b.Created, &b.Updated, &b.StartTime, &b.EndTime, &b.Status, &b.Tag); err != nil {
				return errs.SQLError(err)
			}
			builds = append(builds, b)
		}

		if len(builds) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
}

func (b *buildStore) FindBuildStatus(ctx context.Context,
	repoName,
	orgID,
	branchName string, startDate, endDate time.Time) ([]*core.Build, error) {
	builds := make([]*core.Build, 0)
	return builds, b.db.Execute(func(db *sqlx.DB) error {
		args := []interface{}{repoName, orgID, startDate, endDate}
		query := findBuildStatusQuery
		if branchName != "" {
			args = append(args, branchName)
			query = findBuildStatusBranchQuery
		}
		rows, err := db.Queryx(query, args...)
		if err != nil {
			return errs.SQLError(err)
		}
		if rows.Err() != nil {
			return rows.Err()
		}
		defer rows.Close()
		for rows.Next() {
			b := new(core.Build)
			if err := rows.Scan(&b.Created,
				&b.ExecutionStatus.Total,
				&b.ExecutionStatus.Error,
				&b.ExecutionStatus.Failed,
				&b.ExecutionStatus.Aborted,
				&b.ExecutionStatus.Passed,
				&b.ExecutionStatus.Initiating,
				&b.ExecutionStatus.Running); err != nil {
				return errs.SQLError(err)
			}
			builds = append(builds, b)
		}

		if len(builds) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
}

func (b *buildStore) FindBuildStatusFailed(ctx context.Context,
	repoName,
	orgID,
	branchName string,
	startDate, endDate time.Time,
	limit int) ([]*core.Build, error) {
	builds := make([]*core.Build, 0, limit)
	return builds, b.db.Execute(func(db *sqlx.DB) error {
		args := []interface{}{repoName, orgID, startDate, endDate}
		query := findBuildStatusFailedQuery
		if branchName != "" {
			args = append(args, branchName)
			query = findBuildStatusFailedBranchQuery
		}
		args = append(args, limit)
		rows, err := db.Queryx(query, args...)
		if err != nil {
			return errs.SQLError(err)
		}
		if rows.Err() != nil {
			return rows.Err()
		}
		defer rows.Close()
		for rows.Next() {
			bd := new(core.Build)
			var jsonRaw string
			if err := rows.Scan(&bd.ExecutionStatus.Failed, &bd.Created, &bd.CommitAuthor, &jsonRaw); err != nil {
				return errs.SQLError(err)
			}
			if err := json.Unmarshal([]byte(jsonRaw), &bd.BuildsList); err != nil {
				b.logger.Errorf("failed to unmarshall json for builds repoName %s,orgID %s, error %v", repoName, orgID, err)
				return err
			}
			if len(bd.BuildsList) > 1 {
				sort.Slice(bd.BuildsList, func(i, j int) bool {
					return bd.BuildsList[i].Created > bd.BuildsList[j].Created
				})
			}
			builds = append(builds, bd)
		}

		if len(builds) == 0 {
			return errs.ErrRowsNotFound
		}

		return nil
	})
}
func (b *buildStore) StoreBuildCache(ctx context.Context, buildID string, buildCache *core.BuildCache) error {
	key := utils.GetBuildHashKey(buildID)
	buildCacheMap := map[string]interface{}{
		"token_path":              buildCache.TokenPath,
		"installation_token_path": buildCache.InstallationTokenPath,
		"secret_path":             buildCache.SecretPath,
		"git_provider":            buildCache.GitProvider.String(),
		"repo_slug":               buildCache.RepoSlug,
		"pull_request_number":     buildCache.PullRequestNumber,
		"payload_address":         buildCache.PayloadAddress,
		"base_commit_id":          buildCache.BaseCommitID,
		"target_commit_id":        buildCache.TargetCommitID,
		"repo_id":                 buildCache.RepoID,
		"collect_coverage":        buildCache.CollectCoverage,
		"tier":                    string(buildCache.Tier),
		"tag":                     string(buildCache.BuildTag),
		"branch":                  buildCache.Branch,
		"aborted":                 false,
	}
	// pipeline the commands to reduce 1 RTT
	_, err := b.redisDB.Client().Pipelined(context.Background(), func(pipe redis.Pipeliner) error {
		pipe.HSet(context.Background(), key, buildCacheMap)
		pipe.Expire(context.Background(), key, constants.BuildTimeout)
		return nil
	})
	return err
}

func (b *buildStore) CheckBuildCache(ctx context.Context, buildID string) error {
	key := utils.GetBuildHashKey(buildID)
	exists, err := b.redisDB.Client().Exists(context.Background(), key).Result()
	if exists == 0 || err != nil {
		return errs.ErrRedisKeyNotFound
	}
	return nil
}

func (b *buildStore) GetBuildCache(ctx context.Context, buildID string) (*core.BuildCache, error) {
	var build core.BuildCache
	key := utils.GetBuildHashKey(buildID)

	if err := b.CheckBuildCache(ctx, buildID); err != nil {
		return nil, err
	}
	if err := b.redisDB.Client().HGetAll(context.Background(), key).Scan(&build); err != nil {
		return nil, err
	}
	return &build, nil
}

func (b *buildStore) UpdateExecTaskCount(ctx context.Context, buildID string, count int) error {
	key := utils.GetBuildHashKey(buildID)
	exists, err := b.redisDB.Client().Exists(ctx, key).Result()
	if exists == 0 || err != nil {
		b.logger.Errorf("failed to find key for buildID %s, in redis, error %v", buildID, err)
		return errs.ErrRedisKeyNotFound
	}
	b.logger.Debugf("Incrementing tasks for key: %s by %d", key, count)
	return b.IncrementCacheCount(ctx, buildID, core.TotalExecTasks, count)
}

func (b *buildStore) UpdateFlakyExecutionCount(ctx context.Context, buildID string, executionCount int) error {
	key := utils.GetBuildHashKey(buildID)
	exists, err := b.redisDB.Client().Exists(ctx, key).Result()
	if exists == 0 || err != nil {
		b.logger.Errorf("failed to find key for buildID %s, in redis, error %v", buildID, err)
		return errs.ErrRedisKeyNotFound
	}
	b.logger.Debugf("setting flaky_consecutive_runs for build: %s to %d", key, executionCount)
	_, err = b.redisDB.Client().HSet(ctx, key, map[string]interface{}{
		"flaky_consecutive_runs": executionCount,
	}).Result()
	return err
}

func (b *buildStore) DeleteBuildCache(ctx context.Context, buildID string) error {
	key := utils.GetBuildHashKey(buildID)
	_, err := b.redisDB.Client().Del(ctx, key).Result()
	return err
}

func (b *buildStore) FindTests(ctx context.Context, buildID, repoName, orgID string, offset, limit int) ([]*core.Test, error) {
	tests := make([]*core.Test, 0)
	return tests, b.db.Execute(func(db *sqlx.DB) error {
		args := []interface{}{repoName, orgID, buildID, limit, offset}
		query := findTestQuery
		rows, err := db.QueryxContext(ctx, query, args...)
		if err != nil {
			return errs.SQLError(err)
		}

		defer rows.Close()
		for rows.Next() {
			t := new(core.Test)
			t.Execution = new(core.TestExecution)
			var parentSuiteID, suiteName, suiteDebutCommit zero.String
			if err = rows.Scan(&t.ID,
				&t.Name,
				&t.DebutCommit,
				&t.TestSuiteID,
				&parentSuiteID,
				&suiteName,
				&suiteDebutCommit,
				&t.Execution.CommitID,
				&t.Execution.Duration,
				&t.Execution.Status,
				&t.Execution.Created,
				&t.Execution.StartTime,
				&t.Execution.EndTime,
				&t.Execution.BuildNum,
				&t.Execution.BuildTag); err != nil {
				return errs.SQLError(err)
			}
			if t.TestSuiteID.Valid {
				t.Suite = new(core.TestSuite)
				t.Suite.ID = t.TestSuiteID.String
				t.Suite.ParentSuiteID = parentSuiteID
				t.Suite.Name = suiteName.String
				t.Suite.DebutCommit = suiteDebutCommit.String
			}
			tests = append(tests, t)
		}
		if len(tests) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
}

func (b *buildStore) FindBaseBuildCommitForRebuildPostMerge(ctx context.Context, repoID,
	branch, commitID string) (string, error) {
	var commitSha string
	err := b.db.Execute(func(db *sqlx.DB) error {
		row := db.QueryRowxContext(ctx, findBaseCommitForRebuildQuery, repoID, branch, commitID)
		if err := row.Scan(&commitSha); err != nil {
			return errs.SQLError(err)
		}
		return nil
	})

	if err != nil {
		return "", err
	}
	return commitSha, nil
}

func (b *buildStore) FindLastBuildCommitSha(ctx context.Context, repoID, branch string) (string, error) {
	var commitSha string
	err := b.db.Execute(func(db *sqlx.DB) error {
		row := db.QueryRowxContext(ctx, findLastBuildCommitQuery, repoID, branch, repoID)
		if err := row.Scan(&commitSha); err != nil {
			return errs.SQLError(err)
		}
		return nil
	})

	if err != nil {
		return "", err
	}
	return commitSha, nil
}

func (b *buildStore) FindBuildMeta(ctx context.Context, repoName, orgID, branch string) ([]*core.BuildExecutionStatus, error) {
	buildMeta := make([]*core.BuildExecutionStatus, 0)
	return buildMeta, b.db.Execute(func(db *sqlx.DB) error {
		args := []interface{}{repoName, orgID}
		query := findBuildMetaQuery
		if branch != "" {
			query = fmt.Sprintf(query, " AND b.branch_name = ? ")
			args = append(args, branch)
		} else {
			query = fmt.Sprintf(query, "")
		}
		rows, err := db.QueryxContext(ctx, query, args...)
		if err != nil {
			return errs.SQLError(err)
		}

		defer rows.Close()
		for rows.Next() {
			b := new(core.BuildExecutionStatus)
			if err = rows.Scan(&b.Total,
				&b.Passed,
				&b.Failed,
				&b.Skipped,
				&b.Error,
				&b.Aborted,
				&b.Initiating,
				&b.Running); err != nil {
				return errs.SQLError(err)
			}
			buildMeta = append(buildMeta, b)
		}
		if len(buildMeta) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
}

func (b *buildStore) CheckIfCommitIDBuilt(ctx context.Context, commitID, repoID string) (bool, error) {
	var hasBuild bool
	err := b.db.Execute(func(db *sqlx.DB) error {
		row := db.QueryRowxContext(ctx, findBuildForCommitQuery, commitID, repoID)
		if err := row.Scan(&hasBuild); err != nil {
			return err
		}
		return nil
	})
	return hasBuild, err
}

func (b *buildStore) CountPassedFailedBuildsForRepo(ctx context.Context, repoID string) (int, error) {
	buildCount := 0
	err := b.db.Execute(func(db *sqlx.DB) error {
		rows := db.QueryRowxContext(ctx, countPFBuilds, repoID)
		if err := rows.Scan(&buildCount); err != nil {
			return errs.SQLError(err)
		}
		return nil
	})
	return buildCount, err
}

func (b *buildStore) FindCommitDiff(ctx context.Context, orgName, repoName, gitProvider, buildID string) (string, error) {
	var eventType, baseCommit, commitID string
	var payload json.RawMessage
	var parsedReponse core.Payloadresponse
	err := b.db.Execute(func(db *sqlx.DB) error {
		query := findCommitDiffQuery
		rows := db.QueryRowxContext(ctx, query, buildID, repoName)
		if err := rows.Scan(&payload, &eventType, &baseCommit, &commitID); err != nil {
			return errs.SQLError(err)
		}
		if err := json.Unmarshal(payload, &parsedReponse); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if eventType == string(core.EventPush) {
		if baseCommit == "" {
			return "", errs.ErrRowsNotFound
		}
		commitDiffURL := diffURLPostMerge(gitProvider, orgName, repoName, baseCommit, commitID)
		return commitDiffURL, nil
	} else if eventType == string(core.EventPullRequest) {
		commitDiffURL := diffURLPreMerge(gitProvider, parsedReponse.PullRequests.Link)
		return commitDiffURL, nil
	} else {
		return "", errs.ErrRowsNotFound
	}
}

func (b *buildStore) FindLastBuildTierInTx(ctx context.Context, tx *sqlx.Tx, repoID string) (core.Tier, error) {
	lastBuildTier := new(core.Tier)
	err := b.db.Execute(func(db *sqlx.DB) error {
		row := db.QueryRowxContext(ctx, findLastBuildTier, repoID)
		if err := row.Scan(lastBuildTier); err != nil {
			return err
		}
		return nil
	})
	return *lastBuildTier, err
}

func (b *buildStore) FindMttfAndMttr(ctx context.Context, repoID, branchName string,
	startDate, endDate time.Time) (*core.MttfAndMttr, error) {
	mttfAndMttr := new(core.MttfAndMttr)
	data := make([]*core.MttfAndMttr, 0)
	err := b.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo_id":        repoID,
			"branch":         branchName,
			"min_created_at": startDate,
			"max_created_at": endDate,
		}
		query := listJobsStatus
		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}
		defer rows.Close()
		for rows.Next() {
			newJob := new(core.MttfAndMttr)
			if errS := rows.StructScan(newJob); errS != nil {
				return errs.SQLError(err)
			}
			data = append(data, newJob)
		}
		if len(data) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	mttfAndMttr.MTTF, mttfAndMttr.MTTR = processMttfMttr(data)
	return mttfAndMttr, nil
}

func (b *buildStore) UpdateTestsRuntimes(ctx context.Context, build *core.Build) error {
	return b.db.Execute(func(db *sqlx.DB) error {
		if _, err := db.NamedExecContext(ctx, updateTestRuntimesQuery, build); err != nil {
			return err
		}
		return nil
	})
}

func (b *buildStore) FindBuildWiseTimeSaved(ctx context.Context, repoID, branchName string, minCreatedAt time.Time,
	maxCreatedAt time.Time, limit, offset int) ([]*core.BuildWiseTimeSaved, error) {
	buildWiseTimeSaved := make([]*core.BuildWiseTimeSaved, 0)
	err := b.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo_id":        repoID,
			"min_created_at": minCreatedAt,
			"max_created_at": maxCreatedAt,
			"branch_name":    branchName,
			"limit":          limit,
			"offset":         offset,
		}
		query := buildWiseTimeSavedQuery
		branchWhere := ""
		if branchName != "" {
			branchWhere = " AND b.branch_name = :branch_name"
		}
		query = fmt.Sprintf(query, branchWhere)
		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}
		defer rows.Close()
		for rows.Next() {
			ts := new(core.BuildWiseTimeSaved)
			if errS := rows.StructScan(&ts); errS != nil {
				return errs.SQLError(errS)
			}
			buildWiseTimeSaved = append(buildWiseTimeSaved, ts)
		}
		if rows.Err() != nil {
			return rows.Err()
		}
		if len(buildWiseTimeSaved) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
	if err != nil {
		return nil, errs.SQLError(err)
	}
	return buildWiseTimeSaved, nil
}

func diffURLPreMerge(gitProvider, link string) string {
	if gitProvider == core.DriverGithub.String() {
		commitDiffURL := link + "/files"
		return commitDiffURL
	} else if gitProvider == core.DriverBitbucket.String() {
		commitDiffURL := link + "/diffs"
		return commitDiffURL
	} else {
		commitDiffURL := link
		return commitDiffURL
	}
}

func diffURLPostMerge(gitProvider, orgName, repoName, baseCommit, commitID string) string {
	if gitProvider == core.DriverGithub.String() {
		commitDiffURL := fmt.Sprintf("https://github.com/%s/%s/compare/%s...%s", orgName, repoName, baseCommit, commitID)
		return commitDiffURL
	} else if gitProvider == core.DriverBitbucket.String() {
		commitDiffURL := fmt.Sprintf("https://bitbucket.org/%s/%s/compare/%s..%s", orgName, repoName, commitID, baseCommit)
		return commitDiffURL
	} else {
		commitDiffURL := fmt.Sprintf("https://gitlab.com/%s/%s/-/compare/%s...%s", orgName, repoName, baseCommit, commitID)
		return commitDiffURL
	}
}

func (b *buildStore) UpdateBuildCache(ctx context.Context, buildID string, updateMap map[string]interface{}, skipKeycheck bool) error {
	if !skipKeycheck {
		if err := b.CheckBuildCache(ctx, buildID); err != nil {
			return err
		}
	}
	key := utils.GetBuildHashKey(buildID)
	_, err := b.redisDB.Client().HSet(ctx, key, updateMap).Result()
	return err
}

func (b *buildStore) IncrementCacheCount(ctx context.Context, buildID, field string, testCount int) error {
	key := utils.GetBuildHashKey(buildID)
	if _, err := b.redisDB.Client().HIncrBy(ctx, key, field, int64(testCount)).Result(); err != nil {
		return err
	}
	return nil
}

func processMttfMttr(jobs []*core.MttfAndMttr) (mttf, mttr int64) {
	// nttf, nttr are number of TTF and TTR respectively
	var nttf int64 = 0
	var nttr int64 = 0
	// return -1 for mttf if no pass to fail transition is found
	mttf = -1
	// return -1 for mttr if no fail to pass transition is found
	mttr = -1
	passedJob := core.BuildPassed
	failedJob := core.BuildFailed
	var lastFailedTime, lastPassedTime time.Time
	lastPassedJobExist := false
	lastFailedJobExist := false
	for i := 0; i < len(jobs); i++ {
		if jobs[i].Status == passedJob && !lastPassedJobExist {
			lastPassedTime = jobs[i].TimeStamp
			lastPassedJobExist = true
		} else if jobs[i].Status == failedJob && !lastFailedJobExist {
			lastFailedTime = jobs[i].TimeStamp
			lastFailedJobExist = true
		}
		if i == 0 {
			continue
		}
		prev := jobs[i-1]
		curr := jobs[i]
		if prev.Status == passedJob && curr.Status == failedJob {
			nttf++
			mttf += int64((curr.TimeStamp.Sub(lastPassedTime)).Seconds())
			lastFailedTime = curr.TimeStamp
		}
		if prev.Status == failedJob && curr.Status == passedJob {
			nttr++
			mttr += int64((curr.TimeStamp.Sub(lastFailedTime)).Seconds())
			lastPassedTime = curr.TimeStamp
		}
	}
	if nttf != 0 {
		mttf /= nttf
	}
	if nttr != 0 {
		mttr /= nttr
	}
	return mttf, mttr
}

// FindJobsWithTestsStatus finds the status of latest `limit` builds
func (b *buildStore) FindJobsWithTestsStatus(ctx context.Context,
	repoName,
	orgID,
	branchName string,
	limit int) ([]*core.Build, error) {
	builds := make([]*core.Build, 0, limit)
	return builds, b.db.Execute(func(db *sqlx.DB) error {
		query := listJobsTestsStatusQuery
		args := []interface{}{repoName, orgID}
		branchWhere := ""
		if branchName != "" {
			branchWhere = " AND b.branch_name = ? "
			args = append(args, branchName)
		}
		args = append(args, limit)
		query = fmt.Sprintf(query, branchWhere)
		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			return errs.SQLError(err)
		}
		if rows.Err() != nil {
			return rows.Err()
		}
		defer rows.Close()
		for rows.Next() {
			b := new(core.Build)
			b.Meta = new(core.ExecutionMeta)
			if err := rows.Scan(
				&b.ID,
				&b.Tag,
				&b.Branch,
				&b.Created,
				&b.Meta.Total,
				&b.Meta.Failed,
				&b.Meta.Aborted,
				&b.Meta.Passed,
				&b.Meta.Blocklisted,
				&b.Meta.Quarantined,
				&b.Meta.Skipped); err != nil {
				return errs.SQLError(err)
			}
			builds = append(builds, b)
		}
		if len(builds) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
}

const branchCondition = `AND b.branch_name = :branch`

const createdAtCondition = `AND b.created_at <= :last_seen_time`

const countPFBuilds = `
SELECT
    COUNT(DISTINCT b.id)
FROM
    build b
WHERE
    b.repo_id = ?
    AND b.status IN ('passed', 'failed')
`

const findByBuildIDQuery = `SELECT
    build_num,
	commit_id,
	base_commit,
	event_id,
	repo_id,
	status,
	start_time,
	payload_address,
	tag,
	branch_name
FROM
    build
WHERE 
    id = ?`

const insertQuery = `INSERT
INTO
build(build_num,
	id,
	commit_id,
	base_commit,
	event_id,
	repo_id,
	status,
	start_time,
	payload_address,
	tag,
	branch_name
)
SELECT 1+IFNULL(MAX(build_num),0) AS build_num,
	:id,
	:commit_id,
	:base_commit,
	:event_id,
	:repo_id,
	:status,
	:start_time,
	:payload_address,
	:tag,
	:branch_name
FROM 
    build
WHERE repo_id = :repo_id`

const findByRepoQuery = `
WITH build_meta AS (
	SELECT
	    b.build_num,
		b.id,
		b.created_at build_created_date,
		b.commit_id,
		c.author_name,
		c.message,
		b.updated_at,
		b.created_at,
		b.start_time,
		b.end_time,
		b.status,
		SUM(JSON_LENGTH(cd.test_ids)) total_tests,
		b.tag,
		b.branch_name,
		b.remark,
		r.job_view
	FROM
		build b
	LEFT JOIN repositories r ON
		b.repo_id = r.id
	LEFT JOIN git_commits c ON
		c.commit_id = b.commit_id
		AND c.repo_id = b.repo_id
	LEFT JOIN commit_discovery cd ON
		cd.commit_id = b.commit_id
		AND cd.repo_id = b.repo_id
	WHERE
		r.name = :repo
		AND r.org_id = :org_id
		%s
		%s
		%s
		%s
	GROUP BY b.id
), flaky_meta AS (
    SELECT
		fte.build_id as build_id,
		COUNT(DISTINCT fte.test_id) as impacted_tests,
		COUNT(CASE WHEN fte.status = 'flaky' THEN fte.id END) as flaky_tests,
		COUNT(CASE WHEN fte.status = 'skipped' THEN fte.id END) skipped,
		COUNT(CASE WHEN fte.status = 'blocklisted' THEN fte.id END) blocklisted,
		COUNT(CASE WHEN fte.status = 'aborted' THEN fte.id END) aborted
	FROM 
		flaky_test_execution fte
	GROUP BY fte.build_id      
)
SELECT
    build.build_num,
	build.id,
	build.commit_id,
	build.author_name,
	build.message,
	build.updated_at,
	build.created_at,
	build.start_time,
	build.end_time,
	build.status,
	build.remark,
	build.job_view,
	COUNT(te.id) total_tests_executed,
	IFNULL(build.total_tests, COUNT(te.id)) total_tests,
	SUM(IFNULL(te.duration, 0)) total_duration,
	COUNT(CASE WHEN te.status = 'passed' THEN te.id END) passed,
	COUNT(CASE WHEN te.status = 'failed' THEN te.id END) failed,
	COUNT(CASE WHEN te.status = 'skipped' THEN te.id END) skipped,
	COUNT(CASE WHEN te.status = 'blocklisted' THEN te.id END) blocklisted,
	COUNT(CASE WHEN te.status = 'quarantined' THEN te.id END) quarantined,
	COUNT(CASE WHEN te.status = 'aborted' THEN te.id END) aborted,
	build.tag,
	build.branch_name,
	COALESCE(fm.impacted_tests, 0) flaky_impacted_tests,
	COALESCE(fm.flaky_tests, 0) flaky_tests,
	COALESCE(fm.skipped, 0) flaky_skipped,
	COALESCE(fm.blocklisted, 0) flaky_blocklisted,
	COALESCE(fm.aborted, 0) flaky_aborted 
FROM
	build_meta build
LEFT JOIN test_execution te ON
	te.build_id = build.id
LEFT JOIN flaky_meta fm ON
    fm.build_id = build.id
WHERE 
	build.id != "" 
	%s
	%s
	%s
GROUP BY build.id
ORDER BY build.build_num DESC
LIMIT :limit
`

const findByCommitQuery = `
SELECT
    b.build_num,
	b.id,
	b.created_at,
	b.updated_at,
	b.start_time,
	b.end_time,
	b.status,
	b.tag
FROM
	git_commits c
JOIN build b ON
	b.commit_id = c.commit_id
	AND b.repo_id = c.repo_id
JOIN repositories r ON
	c.repo_id = r.id
WHERE
	r.name = :repo
	AND r.org_id = :org_id
	AND c.commit_id = :commit_id
	%s
	%s
ORDER BY b.created_at DESC LIMIT :limit
`
const updateQuery = `UPDATE
build
SET
status =:status,
start_time =:start_time,
end_time =:end_time,
updated_at =:updated_at
WHERE
id =:id`

const markStoppedQuery = `
UPDATE
	build
SET
	status = :status,
	remark = :remark,
	end_time = :end_time,
	updated_at = :updated_at
WHERE
	id = :id
`

const updateTierQuery = `
UPDATE
	build
SET
	tier = :tier,
	updated_at = :updated_at
WHERE
	id = :id
`

const markStaleStoppedQuery = `
UPDATE
	build
SET
	status = 'error',
	remark = ?,
	updated_at = NOW()
WHERE
	DATE_ADD(created_at, INTERVAL %f HOUR) <= NOW()
	AND status IN ('initiating', 'running');
`

const findBuildStatusQuery = `
SELECT
	DATE(b.created_at) build_creation_date,
	COUNT(b.id),
	COUNT(CASE WHEN b.status = 'error' THEN b.id END) error,
	COUNT(CASE WHEN b.status = 'failed' THEN b.id END) failed,
	COUNT(CASE WHEN b.status = 'aborted' THEN b.id END) aborted,
	COUNT(CASE WHEN b.status = 'passed' THEN b.id END) passed,
	COUNT(CASE WHEN b.status = 'initiating' THEN b.id END) initiating,
	COUNT(CASE WHEN b.status = 'running' THEN b.id END) running
FROM
	build b
JOIN repositories r ON
	r.id = b.repo_id
WHERE
	r.name = ?
	AND r.org_id = ?
	AND b.created_at >= ?
	AND b.created_at <= ?
GROUP BY
	build_creation_date
ORDER BY build_creation_date DESC `

const findBuildStatusBranchQuery = `
SELECT
	DATE(b.created_at) build_creation_date,
	COUNT(b.id),
	COUNT(CASE WHEN b.status = 'error' THEN b.id END) error,
	COUNT(CASE WHEN b.status = 'failed' THEN b.id END) failed,
	COUNT(CASE WHEN b.status = 'aborted' THEN b.id END) aborted,
	COUNT(CASE WHEN b.status = 'passed' THEN b.id END) passed,
	COUNT(CASE WHEN b.status = 'initiating' THEN b.id END) initiating,
	COUNT(CASE WHEN b.status = 'running' THEN b.id END) running
FROM
	build b
JOIN repositories r ON
	r.id = b.repo_id
WHERE
	r.name = ?
	AND r.org_id = ?
	AND b.created_at >= ?
	AND b.created_at <= ?
	AND b.branch_name = ?
GROUP BY
	build_creation_date
ORDER BY build_creation_date DESC`

const findBuildStatusFailedQuery = `
SELECT
	COUNT(b.id) build_counts,
	MAX(b.created_at) latest_build_creation_date,
	ANY_VALUE(gc.author_name) author_name,
	JSON_ARRAYAGG(JSON_OBJECT('build_id', b.id, 'created_at', UNIX_TIMESTAMP(b.created_at), 'build_tag', b.tag)) id
FROM
	build b
JOIN repositories r
ON
	r.id = b.repo_id
JOIN git_commits gc ON
	gc.commit_id = b.commit_id
	AND gc.repo_id = r.id
WHERE
	r.name = ?
	AND r.org_id = ?
	AND b.created_at >= ?
	AND b.created_at <= ?
	AND b.status = "failed"
GROUP BY gc.author_name 
ORDER BY build_counts DESC LIMIT ? `

const findBuildStatusFailedBranchQuery = `
SELECT
	COUNT(b.id) build_counts,
	MAX(b.created_at) latest_build_creation_date,
	ANY_VALUE(gc.author_name) author_name,
	JSON_ARRAYAGG(JSON_OBJECT('build_id', b.id, 'created_at', UNIX_TIMESTAMP(b.created_at), 'build_tag', b.tag)) id
FROM
	build b
JOIN repositories r
ON
	r.id = b.repo_id
JOIN git_commits gc ON
	gc.commit_id = b.commit_id
	AND gc.repo_id = r.id
WHERE
	r.name = ?
	AND r.org_id = ?
	AND b.created_at >= ?
	AND b.created_at <= ?
	AND b.status = "failed"
	AND b.branch_name = ?
GROUP BY gc.author_name 
ORDER BY build_counts DESC LIMIT ? `

const findTestQuery = `
SELECT
	t.id,
	t.name,
	t.debut_commit,
	t.test_suite_id,
	ts.parent_suite_id,
	ts.name,
	ts.debut_commit,
	te.commit_id,
	te.duration,
	te.status,
	te.created_at,
	te.start_time,
	te.end_time,
	b.build_num,
	b.tag
FROM
	build b
JOIN test_execution te ON
	te.build_id = b.id
JOIN test t ON
	te.test_id = t.id
LEFT JOIN test_suite ts ON
	ts.id = t.test_suite_id
JOIN repositories r ON
	t.repo_id = r.id
WHERE
	r.name = ?
	AND r.org_id = ?
	AND b.id = ?
ORDER BY te.test_id DESC, te.created_at DESC 
LIMIT ? OFFSET ?
`
const findBaseCommitForRebuildQuery = `
SELECT 
    COALESCE(base_commit, "") 
FROM 
    build b
WHERE 
    b.repo_id = ?
    AND b.branch_name = ?
    AND b.commit_id = ?
    AND b.tag = "postmerge"
    ORDER BY b.updated_at DESC LIMIT 1		
`
const findLastBuildCommitQuery = `
WITH buildno AS (
	SELECT IFNULL(MAX(b.build_num), 0) AS build_num
	FROM
		build b
	WHERE
		b.repo_id = ?
		AND b.branch_name = ?
		AND b.tag = "postmerge"
		AND b.status = "passed"
	)
SELECT 
    commit_id 
FROM
    build b 
JOIN buildno bno ON 
    b.build_num = bno.build_num 
WHERE b.repo_id = ?`

const findBuildMetaQuery = `
SELECT
	COUNT(DISTINCT b.id) total,
	COUNT(DISTINCT CASE WHEN b.status = 'passed' THEN b.id END) passed,
	COUNT(DISTINCT CASE WHEN b.status = 'failed' THEN b.id END) failed,
	COUNT(DISTINCT CASE WHEN b.status = 'skipped' THEN b.id END) skipped,
	COUNT(DISTINCT CASE WHEN b.status = 'error' THEN b.id END) error,
	COUNT(DISTINCT CASE WHEN b.status = 'aborted' THEN b.id END) aborted,
	COUNT(DISTINCT CASE WHEN b.status = 'initiating' THEN b.id END) initiating,
	COUNT(DISTINCT CASE WHEN b.status = 'running' THEN b.id END) running
FROM
	build b
JOIN repositories r ON
	b.repo_id = r.id
WHERE
	r.name = ?
	AND r.org_id = ?
	%s
`

const findBuildForCommitQuery = `
SELECT
	1
FROM
	build
WHERE
	commit_id = ?
	AND repo_id = ?
LIMIT 1`

// nolint:gosec
const checkIfPassedBuildExists = `
SELECT 1 FROM build b
WHERE 
     b.repo_id = ?
	 AND b.status = 'passed'
LIMIT 1
`
const findCommitDiffQuery = `
SELECT
	COALESCE(g.event_payload, ""),
	g.event_name,
	COALESCE(b.base_commit, ""),
	b.commit_id
FROM
	git_events g
JOIN build b ON
	b.commit_id = g.commit_id
JOIN repositories r ON
	r.id = g.repo_id
WHERE
	b.id = ?
	AND r.name = ?
`

const findLastBuildTier = `
SELECT
	tier
FROM
	build
WHERE
	repo_id = ?
	AND status IN ('passed', 'failed')
ORDER BY created_at DESC
LIMIT 1
`

const listJobsStatus = `
SELECT
	b.status,
	b.created_at
FROM
	build b
WHERE
	b.repo_id = :repo_id
	AND b.branch_name = :branch
	AND b.tag = "postmerge"
	AND b.status IN ("passed", "failed")
	AND b.created_at >= :min_created_at
	AND b.created_at <= :max_created_at
ORDER BY b.created_at 
`
const listJobsTestsStatusQuery = `
SELECT
	b.id,
	b.tag,
	b.branch_name,
	MAX(b.created_at) build_latest_creation_date,
	COUNT(te.test_id),
	COUNT(CASE WHEN te.status = 'failed' THEN te.id END) failed,
	COUNT(CASE WHEN te.status = 'aborted' THEN te.id END) aborted,
	COUNT(CASE WHEN te.status = 'passed' THEN te.id END) passed,
	COUNT(CASE WHEN te.status = 'blocklisted' THEN te.id END) blocklisted,
	COUNT(CASE WHEN te.status = 'quarantined' THEN te.test_id END) quarantined,
	COUNT(CASE WHEN te.status = 'skipped' THEN te.id END) skipped
FROM 
	build b
LEFT JOIN test_execution te ON
	te.build_id = b.id
JOIN repositories r ON
	r.id = b.repo_id
WHERE
	r.name = ?
	AND r.org_id = ?
	%s
GROUP BY
	b.id
ORDER BY
	build_latest_creation_date DESC
LIMIT ? `

const updateTestRuntimesQuery = `
UPDATE
	build
SET
	time_all_tests_ms = :time_all_tests_ms,
	time_impacted_tests_ms = :time_impacted_tests_ms,
	updated_at = :updated_at
WHERE
	id = :id
`

const buildWiseTimeSavedQuery = `
SELECT
	b.id build_id,
	b.time_all_tests_ms,
	b.time_impacted_tests_ms,
	COALESCE((b.time_all_tests_ms - b.time_impacted_tests_ms), 0) time_saved_ms,
	CASE 
		WHEN b.time_all_tests_ms = 0 THEN 0
		ELSE round((( b.time_all_tests_ms - b.time_impacted_tests_ms) / b.time_all_tests_ms * 100 ), 2)
	END percent_time_saved,
	SUM(COALESCE((b.time_all_tests_ms - b.time_impacted_tests_ms), 0)) OVER() net_time_saved_ms
FROM
	build b
WHERE
	b.repo_id = :repo_id
	AND b.created_at >= :min_created_at
	AND b.created_at <= :max_created_at
	AND b.status IN ("passed", "failed")
	AND b.tag IN ("postmerge", "premerge")
	%s
ORDER BY b.build_num DESC
LIMIT :limit
OFFSET :offset
`
