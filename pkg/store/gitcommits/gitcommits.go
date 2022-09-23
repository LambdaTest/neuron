package gitcommits

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/jmoiron/sqlx"
	"gopkg.in/guregu/null.v4/zero"
)

type gitCommitStore struct {
	db     core.DB
	logger lumber.Logger
}

const insertQueryChunkSize = 1000

// New returns a new GitCommitStore
func New(db core.DB, logger lumber.Logger) core.GitCommitStore {
	return &gitCommitStore{db: db, logger: logger}
}

func (g *gitCommitStore) CreateInTx(ctx context.Context, tx *sqlx.Tx, commits []*core.GitCommit) error {
	return utils.Chunk(insertQueryChunkSize, len(commits), func(start int, end int) error {
		if _, err := tx.NamedExecContext(ctx, insertCommits, commits[start:end]); err != nil {
			return err
		}
		return nil
	})
}

func (g *gitCommitStore) FindByCommitID(ctx context.Context, commitID, repoID string) (string, error) {
	var ID string
	err := g.db.Execute(func(db *sqlx.DB) error {
		selectQuery := "SELECT id FROM git_commits WHERE repo_id=? AND commit_id=?"
		rows := db.QueryRowContext(ctx, selectQuery, repoID, commitID)
		if err := rows.Scan(&ID); err != nil {
			return errs.SQLError(err)
		}
		return nil
	})
	return ID, err
}

func (g *gitCommitStore) FindByCommitIDTx(ctx context.Context, tx *sqlx.Tx, commitID, repoID string) (string, error) {
	var ID string
	row := tx.QueryRowxContext(ctx, commitIDFindQuery, repoID, commitID)
	if scanErr := row.Scan(&ID); scanErr != nil {
		g.logger.Errorf("Error in scanning rows, error: %v", scanErr)
		return "", errs.SQLError(scanErr)
	}
	return ID, nil
}

//nolint:goconst
func getFindByRepoQuery(buildID, branchName, commitID, statusFilter, searchText string,
	authorsNames []string, args map[string]interface{}) (query string, queryArgs map[string]interface{}) {
	query = listCommitsByRepoQuery
	queryArgs = args
	branchCommitJoin, branchWhere, commitWhere, statusWhere, buildWhere := "", "", "", "", ""
	textWhere, branchCommitWhere, authorWhere := "", "", ""
	if buildID != "" {
		buildWhere = " AND b.id = :build_id "
	}
	if branchName != "" {
		branchCommitJoin = `JOIN branch_commit bc ON bc.commit_id = gc.commit_id AND bc.repo_id = gc.repo_id`
		branchWhere = `AND b.branch_name = :branch`
		branchCommitWhere = `AND bc.branch_name = :branch`
	}
	if commitID != "" {
		commitWhere = `AND gc.commit_id = :commit_id`
	}
	if statusFilter != "" {
		if statusFilter == "skipped" {
			statusWhere = `AND clb.status IS NULL `
		} else {
			statusWhere = `AND clb.status = :status`
		}
	}
	if searchText != "" {
		textWhere = `AND concat(gc.commit_id, gc.message) LIKE :search_text`
	}
	if len(authorsNames) > 0 {
		authorWhere = `AND gc.author_name IN (%s)`
		inClause := ""
		inClause, queryArgs = utils.AddMultipleFilters("authorNames", authorsNames, queryArgs)
		authorWhere = fmt.Sprintf(authorWhere, inClause)
	}
	query = fmt.Sprintf(query, buildWhere, branchWhere, buildWhere, branchWhere,
		buildWhere, branchWhere, buildWhere, branchWhere, branchCommitJoin, branchCommitWhere, commitWhere, textWhere, statusWhere, authorWhere)
	query += `LIMIT :limit OFFSET :offset`
	return query, queryArgs
}

// FindByRepo find the tests by repo
func (g *gitCommitStore) FindByRepo(ctx context.Context,
	repoName, orgID, commitID, buildID, branchName, orgName, gitProvider, statusFilter, searchText string, authorsNames []string,
	offset, limit int) ([]*core.GitCommit, error) {
	commits := make([]*core.GitCommit, 0, limit)
	return commits, g.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo":        repoName,
			"org_id":      orgID,
			"branch":      branchName,
			"commit_id":   commitID,
			"status":      statusFilter,
			"search_text": "%" + searchText + "%",
			"limit":       limit,
			"offset":      offset,
			"build_id":    buildID,
		}
		query, args := getFindByRepoQuery(buildID, branchName, commitID, statusFilter, searchText, authorsNames, args)

		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}
		if rows.Err() != nil {
			return rows.Err()
		}
		defer rows.Close()
		for rows.Next() {
			c := new(core.GitCommit)
			c.Meta = new(core.TaskMeta)
			c.TestMeta = new(core.ExecutionMeta)
			c.FlakyMeta = new(core.FlakyExecutionMetadata)
			var baseCommit string
			var payload json.RawMessage
			var latestBuildNum int
			var latestBuildID, latestBuildStatus, latestBuildTag, latestBuildRemark, latestBuildBranch zero.String
			var latestBuildCreated, latestBuildStartTime, latestBuildEndTime zero.Time
			if err = rows.Scan(&c.CommitID,
				&c.ParentCommitID, &c.Message, &c.Link, &c.Author.Name,
				&c.Author.Email, &c.Author.Date, &c.Committer.Name, &c.Committer.Email,
				&c.Committer.Date, &c.TestsAdded, &baseCommit, &payload, &latestBuildNum, &latestBuildID,
				&latestBuildCreated, &latestBuildStartTime, &latestBuildEndTime,
				&latestBuildStatus, &latestBuildTag, &latestBuildBranch, &latestBuildRemark,
				&c.Meta.TotalBuilds, &c.Meta.AvgTaskDuration, &c.TestMeta.Completed,
				&c.TestMeta.Aborted, &c.TestMeta.Failed, &c.TestMeta.Blocklisted,
				&c.TestMeta.Quarantined, &c.TestMeta.Skipped, &c.TestMeta.Passed,
				&c.FlakyMeta.ImpactedTests, &c.FlakyMeta.FlakyTests, &c.FlakyMeta.Skipped,
				&c.FlakyMeta.Blocklisted,
				&c.FlakyMeta.Aborted,
			); err != nil {
				return errs.SQLError(err)
			}
			if c.FlakyMeta.ImpactedTests == 0 {
				c.FlakyMeta.OverAllFlakiness = 0
			} else {
				c.FlakyMeta.OverAllFlakiness = float32(c.FlakyMeta.FlakyTests) / float32(c.FlakyMeta.ImpactedTests)
			}
			if latestBuildID.Valid {
				c.LatestBuild = &core.Build{
					BuildNum:  latestBuildNum,
					ID:        latestBuildID.String,
					Created:   latestBuildCreated.Time,
					StartTime: latestBuildStartTime,
					EndTime:   latestBuildEndTime,
					Status:    core.BuildStatus(latestBuildStatus.String),
					Tag:       core.BuildTag(latestBuildTag.String),
					Remark:    latestBuildRemark,
					Branch:    latestBuildBranch.String,
				}
			}
			if c.CommitDiffURL, err = getDiffURL(latestBuildTag.String, gitProvider, orgName,
				repoName, baseCommit, commitID, payload); err != nil {
				return err
			}
			commits = append(commits, c)
		}
		if len(commits) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
}

// FindImpactedTests find the tests which are impacted
func (g *gitCommitStore) FindImpactedTests(ctx context.Context,
	commitID,
	buildID,
	taskID,
	repoName,
	orgID,
	statusFilter,
	searchText, branchName string, offset, limit int) ([]*core.TestExecution, error) {
	tests := make([]*core.TestExecution, 0, limit)
	return tests, g.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo":        repoName,
			"org_id":      orgID,
			"commit_id":   commitID,
			"build_id":    buildID,
			"status":      statusFilter,
			"search_text": "%" + searchText + "%",
			"limit":       limit,
			"offset":      offset,
			"branch":      branchName,
			"task_id":     taskID,
		}
		query := findImpactedTestsQuery
		statusWhere := ""
		searchWhere := ""
		branchBuildJoin := ""
		branchBuildWhere := ""
		taskIDWhere := ""
		if statusFilter != "" {
			statusWhere = `AND te.status = :status`
		}
		if searchText != "" {
			searchWhere = `AND t.name LIKE :search_text`
		}
		if taskID != "" {
			taskIDWhere = ` AND te.task_id = :task_id`
		}
		if branchName != "" {
			branchBuildJoin = `JOIN build b ON b.id = te2.build_id`
			branchBuildWhere = `AND b.branch_name = :branch`
		}
		query = fmt.Sprintf(query, branchBuildJoin, branchBuildWhere, statusWhere, searchWhere, taskIDWhere)
		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}
		if rows.Err() != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			t := new(core.TestExecution)
			t.Transition = new(core.Transition)
			var jsonRaw string
			var statuses []*core.TestStatus
			if err = rows.Scan(&t.TestID,
				&t.TestName,
				&t.TestSuiteName,
				&t.TaskID,
				&t.BuildID,
				&t.Duration,
				&t.Created,
				&t.StartTime,
				&t.EndTime,
				&t.Status,
				&t.CommitAuthor,
				&t.SuiteID,
				&jsonRaw); err != nil {
				return errs.SQLError(err)
			}
			if err := json.Unmarshal([]byte(jsonRaw), &statuses); err != nil {
				g.logger.Errorf("failed to unmarshall json for testID %s, repoName %s,orgID %s, error %v", t.ID, repoName, orgID, err)
				return err
			}
			if len(statuses) > 1 {
				sort.Slice(statuses, func(i, j int) bool {
					return statuses[i].Created > statuses[j].Created
				})
				t.Transition.CurrentStatus = statuses[0].Status.String
				t.Transition.PreviousStatus = statuses[1].Status.String
				if t.Status != core.TestExecutionStatus(t.Transition.CurrentStatus) && statuses[0].Created == statuses[1].Created {
					t.Transition.PreviousStatus = t.Transition.CurrentStatus
					t.Transition.CurrentStatus = string(t.Status)
				}
			} else if len(statuses) == 1 {
				t.Transition.CurrentStatus = statuses[0].Status.String
				t.Transition.PreviousStatus = ""
			} else {
				t.Transition.CurrentStatus = ""
				t.Transition.PreviousStatus = ""
			}
			tests = append(tests, t)
		}
		if len(tests) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
}

func (g *gitCommitStore) FindAuthors(ctx context.Context, repoName, orgID,
	branchName, author, nextAuthor string, limit int) ([]*core.GitCommit, error) {
	commits := make([]*core.GitCommit, 0, limit)
	return commits, g.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo":        repoName,
			"org_id":      orgID,
			"limit":       limit,
			"next_author": nextAuthor,
			"branch":      branchName,
			"author":      "%" + author + "%",
		}
		authorCond := " WHERE ad.author_name LIKE :author "
		nextAuthorCond := " AND ad.author_name <= :next_author "
		query := findAuthorDataQuery
		if branchName != "" {
			query = fmt.Sprintf(query, " JOIN test_branch tb ON tb.repo_id = r.id AND tb.test_id = t.id ",
				" AND tb.branch_name = :branch ", " AND b.branch_name = :branch ", " AND b.branch_name = :branch ",
				` JOIN branch_commit bc ON bc.commit_id = gc.commit_id AND bc.repo_id = r.id `, "AND bc.branch_name = :branch")
		} else {
			query = fmt.Sprintf(query, "", "", "", "", "", "")
		}
		if author != "" && nextAuthor == "" {
			query += authorCond
		} else if author != "" && nextAuthor != "" {
			query += authorCond
			query += nextAuthorCond
		} else if nextAuthor != "" {
			query += " WHERE ad.author_name <= :next_author "
		}
		query += " GROUP BY ad.author_name ORDER BY ad.author_name LIMIT :limit"

		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}
		defer rows.Close()
		for rows.Next() {
			c := new(core.GitCommit)
			c.TestMeta = new(core.ExecutionMeta)
			if err = rows.Scan(&c.TestMeta.Total,
				&c.TestMeta.Passed,
				&c.TestMeta.Failed,
				&c.TestMeta.Skipped,
				&c.TestMeta.Blocklisted,
				&c.TestMeta.Quarantined,
				&c.TestMeta.Aborted,
				&c.Author.Name,
				&c.Author.Date,
				&c.BuildNum,
				&c.BuildID,
				&c.BuildTag,
				&c.Branch,
				&c.Status); err != nil {
				return errs.SQLError(err)
			}
			commits = append(commits, c)
		}
		if len(commits) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
}

func (g *gitCommitStore) FindAuthorCommitActivity(ctx context.Context, repoName, orgID, nextAuthor string, limit int) (map[string]map[int]int, string, error) {
	author := make(map[string]map[int]int)
	var lastSeenAuthor string
	return author, lastSeenAuthor, g.db.Execute(func(db *sqlx.DB) error {
		args := []interface{}{repoName, orgID}
		query := "SELECT c.author_name, JSON_ARRAYAGG(UNIX_TIMESTAMP(DATE(c.authored_date))) FROM git_commits c JOIN repositories r ON r.id=c.repo_id WHERE r.name=? AND r.org_id=? AND c.authored_date  >= NOW() + INTERVAL -1 MONTH "
		if nextAuthor != "" {
			query += " AND c.author_name <= ? "
			args = append(args, nextAuthor)
		}
		query += " GROUP BY c.author_name ORDER BY c.author_name DESC LIMIT ?"
		args = append(args, limit)

		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			return errs.SQLError(err)
		}
		defer rows.Close()
		for rows.Next() {
			var authorName, jsonRaw string
			var timestamps []int

			if err = rows.Scan(&authorName, &jsonRaw); err != nil {
				return errs.SQLError(err)
			}
			if err := json.Unmarshal([]byte(jsonRaw), &timestamps); err != nil {
				g.logger.Errorf("failed to unmarshall json into timestamps array repoName %s,orgID %s, error %v", repoName, orgID, err)
				return err
			}
			frequency := make(map[int]int)
			for _, item := range timestamps {
				if frequency[item] == 0 {
					frequency[item] = 1
				} else {
					frequency[item]++
				}
			}
			author[authorName] = frequency
		}
		if len(author) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
}

// FindByBuild find the commits based on build ID
func (g *gitCommitStore) FindByBuild(ctx context.Context,
	repoName,
	orgID,
	buildID string, offset, limit int) ([]*core.GitCommit, error) {
	commits := make([]*core.GitCommit, 0, limit)
	return commits, g.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo":     repoName,
			"org_id":   orgID,
			"build_id": buildID,
			"limit":    limit,
			"offset":   offset,
		}
		query := listCommitsByBuildQuery
		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}
		if rows.Err() != nil {
			return rows.Err()
		}
		defer rows.Close()
		for rows.Next() {
			var buildStatus, buildRemark zero.String
			c := new(core.GitCommit)
			if err = rows.Scan(&c.CommitID,
				&buildRemark,
				&buildStatus); err != nil {
				return errs.SQLError(err)
			}
			c.LatestBuild = &core.Build{
				Status: core.BuildStatus(buildStatus.String),
				Remark: buildRemark,
			}
			commits = append(commits, c)
		}
		if len(commits) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
}

// FindCommitStatus finds the status of latest `limit` commits
func (g *gitCommitStore) FindCommitStatus(ctx context.Context,
	repoName,
	orgID,
	branchName string,
	limit int) ([]*core.GitCommit, error) {
	commits := make([]*core.GitCommit, 0, limit)
	return commits, g.db.Execute(func(db *sqlx.DB) error {
		query := findCommitStatusQuery
		args := []interface{}{repoName, orgID}
		if branchName != "" {
			query = findCommitStatusBranchQuery
			args = append(args, branchName)
		}
		args = append(args, limit)
		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			return errs.SQLError(err)
		}
		if rows.Err() != nil {
			return rows.Err()
		}
		defer rows.Close()
		for rows.Next() {
			c := new(core.GitCommit)
			c.Coverage = new(core.TestCoverage)
			c.TestMeta = new(core.ExecutionMeta)
			if err := rows.Scan(
				&c.CommitID,
				&c.Coverage.Created,
				&c.Coverage.Blob,
				&c.Coverage.TotalCoverage,
				&c.TestMeta.Total,
				&c.TestMeta.Failed,
				&c.TestMeta.Aborted,
				&c.TestMeta.Passed,
				&c.TestMeta.Blocklisted,
				&c.TestMeta.Quarantined,
				&c.TestMeta.Skipped); err != nil {
				return errs.SQLError(err)
			}
			commits = append(commits, c)
		}
		if len(commits) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
}

// FindCommitMeta find the meta of commits of repo
func (g *gitCommitStore) FindCommitMeta(ctx context.Context,
	repoName, orgID, branchName string) ([]*core.TaskMeta, error) {
	commits := make([]*core.TaskMeta, 0)
	return commits, g.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo":   repoName,
			"org_id": orgID,
			"branch": branchName,
		}
		query := findCommitMetaQuery
		branchCommitWhere, branchCommitJoin, branchBuildJoin := "", "", ""
		if branchName != "" {
			branchCommitJoin = ` JOIN branch_commit bc ON
			bc.commit_id = gc.commit_id
			AND bc.repo_id = gc.repo_id `
			branchCommitWhere = ` AND bc.branch_name = :branch `
			branchBuildJoin = ` AND b.branch_name = :branch `
		}
		query = fmt.Sprintf(query, branchBuildJoin, branchCommitJoin, branchCommitWhere)
		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}
		if rows.Err() != nil {
			return rows.Err()
		}
		defer rows.Close()
		for rows.Next() {
			c := new(core.TaskMeta)
			if err = rows.Scan(&c.Total,
				&c.Passed,
				&c.Failed,
				&c.Skipped,
				&c.Error,
				&c.Aborted,
				&c.Initiating,
				&c.Running); err != nil {
				return errs.SQLError(err)
			}
			skippedCommits := c.Total - c.Passed - c.Failed - c.Error - c.Aborted - c.Initiating - c.Running
			c.Skipped += skippedCommits
			commits = append(commits, c)
		}
		if len(commits) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
}

// FindContributorGraph find the graph of contributors of repo
func (g *gitCommitStore) FindContributorGraph(ctx context.Context,
	repoName, orgID, branchName string) ([]*core.GitCommit, error) {
	graphs := make([]*core.GitCommit, 0)
	return graphs, g.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo":   repoName,
			"org_id": orgID,
			"branch": branchName,
		}
		query := findContributorGraphQuery
		branchCommitWhere := ""
		if branchName != "" {
			branchCommitWhere = ` AND bc.branch_name = :branch `
		}
		query = fmt.Sprintf(query, branchCommitWhere)
		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}
		if rows.Err() != nil {
			return rows.Err()
		}
		defer rows.Close()
		for rows.Next() {
			c := new(core.GitCommit)
			var jsonRaw string
			if err = rows.Scan(
				&c.Author.Name,
				&jsonRaw); err != nil {
				return errs.SQLError(err)
			}
			freqCount := make([]int, constants.MonthInterval)
			var relativeBuildsArray []int
			if err := json.Unmarshal([]byte(jsonRaw), &relativeBuildsArray); err != nil {
				g.logger.Errorf("failed to unmarshall json into days array orgID %s, error %v", orgID, err)
				return err
			}
			for _, value := range relativeBuildsArray {
				if value <= (constants.MonthInterval) && value > 0 {
					freqCount[value-1]++
				}
			}
			c.Graph = freqCount
			graphs = append(graphs, c)
		}
		if len(graphs) == 0 {
			return errs.ErrRowsNotFound
		}
		return nil
	})
}

func getCurrentTransition(orgID, jsonRaw string, g *gitCommitStore) (int, error) {
	var transition int
	var transitionArray []string
	if err := json.Unmarshal([]byte(jsonRaw), &transitionArray); err != nil {
		g.logger.Errorf("failed to unmarshall json into status array orgID %s, error %v", orgID, err)
		return transition, err
	}
	for i := 0; i < len(transitionArray)-1; i++ {
		if transitionArray[i] != transitionArray[i+1] {
			transition++
		}
	}
	return transition, nil
}

// FindAuthorStats finds the stats of authors of a repo
func (g *gitCommitStore) FindAuthorStats(ctx context.Context,
	repoName,
	orgID,
	branchName,
	author string,
	startDate,
	endDate time.Time) (*core.AuthorStats, error) {
	authorStats := &core.AuthorStats{}
	return authorStats, g.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo":       repoName,
			"org":        orgID,
			"author":     author,
			"branch":     branchName,
			"start_date": startDate,
			"end_date":   endDate,
		}
		branchCommitWhere := ""
		if branchName != "" {
			branchCommitWhere = `AND bc.branch_name = :branch`
		}
		query := findAuthorStatsQuery
		query = fmt.Sprintf(query, branchCommitWhere, branchCommitWhere, branchCommitWhere)
		rows, err := db.NamedQueryContext(ctx, query, args)
		if err != nil {
			return errs.SQLError(err)
		}
		if rows.Err() != nil {
			return rows.Err()
		}
		defer rows.Close()
		for rows.Next() {
			if err := rows.StructScan(authorStats); err != nil {
				return err
			}
		}
		return nil
	})
}

// FindTransitions finds the number of transitions of the tests in a particular duration
func (g *gitCommitStore) FindTransitions(ctx context.Context,
	repoName,
	orgID,
	branchName,
	author string,
	startDate,
	endDate time.Time) (int, error) {
	transitions := 0
	return transitions, g.db.Execute(func(db *sqlx.DB) error {
		branchCommitWhere := ""
		argsTransition := map[string]interface{}{
			"repo":       repoName,
			"org_id":     orgID,
			"author":     author,
			"branch":     branchName,
			"start_date": startDate,
			"end_date":   endDate,
		}
		transitionQuery := findTransitionQuery
		if branchName != "" {
			branchCommitWhere = `AND bc.branch_name = :branch`
		}
		transitionQuery = fmt.Sprintf(transitionQuery, branchCommitWhere)
		rows, err := db.NamedQueryContext(ctx, transitionQuery, argsTransition)
		if err != nil {
			return errs.SQLError(err)
		}
		if rows.Err() != nil {
			return rows.Err()
		}
		defer rows.Close()
		for rows.Next() {
			var jsonRaw string
			if err = rows.Scan(&jsonRaw); err != nil {
				return errs.SQLError(err)
			}
			currTransition, err := getCurrentTransition(orgID, jsonRaw, g)
			if err != nil {
				return err
			}
			if rows.Err() != nil {
				return rows.Err()
			}
			transitions += currTransition
		}
		return nil
	})
}

func getDiffURL(latestBuildTag, gitProvider, orgName, repoName, baseCommit,
	commitID string, payload json.RawMessage) (string, error) {
	var parsedResponse core.Payloadresponse
	if err := json.Unmarshal(payload, &parsedResponse); err != nil {
		return "", err
	}
	if latestBuildTag == string(core.PostMergeTag) && baseCommit != "" {
		return diffURLPostMerge(gitProvider, orgName, repoName, baseCommit, commitID)
	}

	if parsedResponse.PullRequests != nil {
		return diffURLPreMerge(gitProvider, parsedResponse.PullRequests.Link)
	}
	return "", nil
}

func diffURLPreMerge(gitProvider, link string) (commitDiffURL string, err error) {
	switch gitProvider {
	case core.DriverGithub.String():
		commitDiffURL = link + "/files"
	case core.DriverBitbucket.String():
		commitDiffURL = link + "/diffs"
	case core.DriverGitlab.String():
		commitDiffURL = link
	default:
		err = errs.ErrUnsupportedGitProvider
	}
	return commitDiffURL, err
}

func diffURLPostMerge(gitProvider, orgName, repoName, baseCommit, commitID string) (commitDiffURL string, err error) {
	switch gitProvider {
	case core.DriverGithub.String():
		commitDiffURL = fmt.Sprintf("https://github.com/%s/%s/compare/%s...%s", orgName, repoName, baseCommit, commitID)
	case core.DriverBitbucket.String():
		commitDiffURL = fmt.Sprintf("https://bitbucket.org/%s/%s/compare/%s..%s", orgName, repoName, commitID, baseCommit)
	case core.DriverGitlab.String():
		commitDiffURL = fmt.Sprintf("https://gitlab.com/%s/%s/-/compare/%s...%s", orgName, repoName, baseCommit, commitID)
	default:
		err = errs.ErrUnsupportedGitProvider
	}
	return commitDiffURL, err
}

const commitIDFindQuery = `SELECT id FROM git_commits WHERE repo_id=? AND commit_id=?`

const insertCommits = `INSERT
INTO
git_commits(id,
commit_id,
author_name,
author_email,
authored_date,
committer_name,
committer_email,
committed_date,
parent_commit_id,
repo_id,
message,
link,
updated_at)
VALUES (:id,
:commit_id,
:author.name,
:author.email,
:author.date,
:committer.name,
:committer.email,
:committer.date,
:parent_commit_id,
:repo_id,
:message,
:link,
:updated_at) ON
DUPLICATE KEY
UPDATE
parent_commit_id =
VALUES(parent_commit_id),
updated_at =
VALUES(updated_at)`

const listCommitsByRepoQuery = `
WITH commits_latest_build AS (
	SELECT
		*
	FROM (
		SELECT
			b.build_num,
			b.id,
			b.status,
			b.commit_id,
			COALESCE(b.base_commit, "") base_commit,
			b.repo_id,
			b.remark,
			b.created_at,
			b.start_time,
			b.end_time,
			b.tag,
			b.branch_name,
			COALESCE(g.event_payload, "") event_payload,
			ROW_NUMBER() OVER (PARTITION BY b.commit_id ORDER BY b.build_num DESC) rn
		FROM
			build b
		JOIN repositories r ON
			r.id = b.repo_id
		JOIN git_events g ON
			b.event_id = g.id
		WHERE
			r.name = :repo
			AND r.org_id = :org_id
			%s
			%s
	) ranked_commit_builds
	WHERE
		rn = 1
),
commits_meta_info AS (
	SELECT
		gc.commit_id,
		COUNT(DISTINCT test.id) tests_added,
		COUNT(DISTINCT b.id) total_builds,
		AVG(IFNULL(TIME_TO_SEC(TIMEDIFF(b.end_time, b.start_time)), 0)) avg_duration,
		COUNT(DISTINCT CASE WHEN b.status = 'initiating' THEN b.id END) initiating,
		COUNT(DISTINCT CASE WHEN b.status = 'running' THEN b.id END) running,
		COUNT(DISTINCT CASE WHEN b.status = 'failed' THEN b.id END) failed,
		COUNT(DISTINCT CASE WHEN b.status = 'aborted' THEN b.id END) aborted,
		COUNT(DISTINCT CASE WHEN b.status = 'passed' THEN b.id END) passed,
		COUNT(DISTINCT CASE WHEN b.status = 'error' THEN b.id END) error
	FROM
		git_commits gc
	JOIN repositories r ON
		gc.repo_id = r.id
	LEFT JOIN test ON
		gc.commit_id = test.debut_commit
		AND gc.repo_id = test.repo_id
	LEFT JOIN build b ON
		gc.commit_id = b.commit_id
		AND gc.repo_id = b.repo_id
	WHERE
		r.name = :repo
		AND r.org_id = :org_id
		%s
		%s
	GROUP BY gc.commit_id
),
commits_tests_info AS (
	SELECT
		gc.commit_id,
		COUNT(DISTINCT CASE WHEN te.status = 'completed' THEN te.test_id END) completed,
		COUNT(DISTINCT CASE WHEN te.status = 'aborted' THEN te.test_id END) aborted,
		COUNT(DISTINCT CASE WHEN te.status = 'failed' THEN te.test_id END) failed,
		COUNT(DISTINCT CASE WHEN te.status = 'blocklisted' THEN te.test_id END) blocklisted,
		COUNT(DISTINCT CASE WHEN te.status = 'quarantined' THEN te.test_id END) quarantined,
		COUNT(DISTINCT CASE WHEN te.status = 'skipped' THEN te.test_id END) skipped,
		COUNT(DISTINCT CASE WHEN te.status = 'passed' THEN te.test_id END) passed
	FROM
		git_commits gc
	JOIN repositories r ON
		gc.repo_id = r.id
	JOIN build b ON
		b.commit_id = gc.commit_id
		AND b.repo_id = r.id
	JOIN test_execution te ON
		b.id = te.build_id
	WHERE
		r.name = :repo
		AND r.org_id = :org_id
		%s
		%s
	GROUP BY gc.commit_id
),
commits_flaky_meta AS (
SELECT
	gc.commit_id,
	COUNT(DISTINCT(fte.test_id)) as impacted_tests,
	COUNT(DISTINCT CASE WHEN fte.status = 'flaky' THEN fte.id END)  flaky_tests,
	COUNT(DISTINCT CASE WHEN fte.status = 'skipped' THEN fte.id END) skipped,
	COUNT(DISTINCT CASE WHEN fte.status = 'blocklisted' THEN fte.id END) blocklisted,
	COUNT(DISTINCT CASE WHEN fte.status = 'aborted' THEN fte.id END) aborted
	FROM
	git_commits gc
JOIN repositories r ON
	gc.repo_id = r.id
JOIN build b ON
	b.commit_id = gc.commit_id
	AND b.repo_id = r.id
JOIN flaky_test_execution fte ON
	b.id = fte.build_id
WHERE
	r.name = :repo
	AND r.org_id = :org_id
	%s
	%s
GROUP BY gc.commit_id      
)
SELECT
	gc.commit_id,
	gc.parent_commit_id,
	gc.message,
	gc.link,
	gc.author_name,
	gc.author_email,
	gc.authored_date,
	gc.committer_name,
	gc.committer_email,
	gc.committed_date,
	COALESCE(cm.tests_added, 0) tests_added,
	COALESCE(clb.base_commit, "") base_commit,
	COALESCE(clb.event_payload, "{}") event_payload,
	COALESCE(clb.build_num, 0) latest_build_num,
	clb.id latest_build_id,
	clb.created_at latest_build_created_at,
	clb.start_time latest_build_start_time,
	clb.end_time latest_build_end_time,
	clb.status latest_build_status,
	clb.tag latest_build_tag,
	clb.branch_name latest_branch_name,
	clb.remark,
	COALESCE(cm.total_builds, 0) total_builds,
	COALESCE(cm.avg_duration*1000, 0) avg_duration,
	COALESCE(cti.completed, 0) tests_completed,
	COALESCE(cti.aborted, 0) tests_aborted,
	COALESCE(cti.failed, 0) tests_failed,
	COALESCE(cti.blocklisted, 0) tests_blocklisted,
	COALESCE(cti.quarantined, 0) tests_quarantined,
	COALESCE(cti.skipped, 0) tests_skipped,
	COALESCE(cti.passed, 0) tests_passed,
	COALESCE(cfm.impacted_tests, 0) flaky_impacted_tests,
	COALESCE(cfm.flaky_tests, 0) flaky_tests,
	COALESCE(cfm.skipped, 0) flaky_skipped,
	COALESCE(cfm.blocklisted, 0) flaky_blocklisted,
	COALESCE(cfm.aborted, 0) flaky_aborted 
FROM
	git_commits gc
%s
JOIN repositories r ON
	r.id = gc.repo_id
LEFT JOIN commits_latest_build clb ON
	gc.commit_id = clb.commit_id
	AND gc.repo_id = clb.repo_id
LEFT JOIN commits_meta_info cm ON
	gc.commit_id = cm.commit_id
LEFT JOIN commits_tests_info cti ON
	gc.commit_id = cti.commit_id
LEFT JOIN commits_flaky_meta cfm ON
    gc.commit_id = cfm.commit_id	
WHERE
	r.name = :repo
	AND r.org_id = :org_id
	%s
	%s
	%s
	%s
	%s
ORDER BY
	gc.authored_date DESC
`

const findImpactedTestsQuery = `
WITH test_authors AS (
	SELECT
		t.id test_id,
		gc.author_name
	FROM
		test t
	JOIN repositories r ON
		t.repo_id = r.id
	JOIN git_commits gc ON
		t.debut_commit = gc.commit_id
		AND gc.repo_id = r.id
	WHERE
		r.name = :repo
		AND r.org_id = :org_id
),
build_time AS (
	SELECT
		COALESCE (MAX(te.created_at),TIMESTAMP("0001-01-01")) max_time,
		ANY_VALUE(r.id) repo_id
	FROM
		test_execution te
	JOIN build b ON
		b.id = te.build_id
	JOIN repositories r ON
		b.repo_id = r.id
	WHERE
		b.id = :build_id
),
ranked_transitions AS (
	SELECT
		te2.test_id ts_id,
		te2.status,
		te2.created_at,
		ROW_NUMBER() OVER (PARTITION BY te2.test_id
	ORDER BY
		te2.updated_at DESC) rn
	FROM
		test_execution te2
	JOIN test t ON
		t.id = te2.test_id
	JOIN repositories r ON
		t.repo_id = r.id
	JOIN build_time bt ON
		bt.repo_id = r.id
		%s
	WHERE
		r.name = :repo
		AND r.org_id = :org_id
		AND te2.created_at <= bt.max_time
		%s
	),
transitions AS (
	SELECT
		rs.ts_id ts_id,
		JSON_ARRAYAGG(
			JSON_OBJECT(
				'status',
				rs.status,
				'created_at',
				UNIX_TIMESTAMP(rs.created_at)
			)
		) transitions
	FROM
		ranked_transitions rs
	WHERE
		rs.rn <= 2
	GROUP BY
		ts_id )
SELECT
	t.id,
	t.name,
	ts.name,
	te.task_id,
	te.build_id,
	te.duration,
	te.created_at,
	te.start_time,
	te.end_time,
	te.status,
	ta.author_name,
	t.test_suite_id,
	COALESCE(trs.transitions, JSON_ARRAY())
FROM
	git_commits c
JOIN build b ON
	b.commit_id = c.commit_id
	AND b.repo_id = c.repo_id
JOIN test_execution te ON
	te.build_id = b.id
JOIN test t ON
	te.test_id = t.id
LEFT JOIN test_suite ts ON
	ts.id = t.test_suite_id
JOIN repositories r ON
	c.repo_id = r.id
JOIN test_authors ta ON
	t.id = ta.test_id
LEFT JOIN transitions trs ON
	trs.ts_id = t.id
WHERE
	r.name = :repo
	AND r.org_id = :org_id
	AND c.commit_id = :commit_id
	AND te.build_id = :build_id
	%s
	%s
	%s
ORDER BY
	te.duration DESC, te.created_at DESC
LIMIT :limit OFFSET :offset
`
const listCommitsByBuildQuery = `
SELECT
	c.commit_id,
	b.remark,
	b.status
FROM
	git_commits c
JOIN repositories r ON
	c.repo_id = r.id
JOIN build b ON
	b.commit_id = c.commit_id
	AND b.repo_id = r.id
WHERE
	r.name = :repo
	AND r.org_id = :org_id
	AND b.id = :build_id
ORDER BY c.authored_date DESC
LIMIT :limit OFFSET :offset
`

const findCommitStatusQuery = `
SELECT
	gc.commit_id,
	MAX(gc.created_at) commit_latest_creation_date,
	COALESCE(ANY_VALUE(tc.blob_link), "") blob_link,
	COALESCE(ANY_VALUE(tc.total_coverage), JSON_OBJECT()) total_coverage,
	COUNT(te.test_id),
	COUNT(CASE WHEN te.status = 'failed' THEN te.id END) failed,
	COUNT(CASE WHEN te.status = 'aborted' THEN te.id END) aborted,
	COUNT(CASE WHEN te.status = 'passed' THEN te.id END) passed,
	COUNT(CASE WHEN te.status = 'blocklisted' THEN te.id END) blocklisted,
	COUNT(CASE WHEN te.status = 'quarantined' THEN te.test_id END) quarantined,
	COUNT(CASE WHEN te.status = 'skipped' THEN te.id END) skipped
FROM
	git_commits gc
JOIN build b ON
	b.commit_id = gc.commit_id
LEFT JOIN test_execution te ON
	te.build_id = b.id
JOIN repositories r ON
	r.id = gc.repo_id
LEFT JOIN test_coverage tc ON
	tc.commit_id = gc.commit_id
	AND tc.repo_id = r.id
WHERE
	r.name = ?
	AND r.org_id = ?
	AND b.status IN ('passed', 'failed', 'error')
GROUP BY
	gc.commit_id
ORDER BY
	commit_latest_creation_date DESC
LIMIT ? `

const findCommitStatusBranchQuery = `
SELECT
	gc.commit_id,
	MAX(gc.created_at) commit_latest_creation_date,
	COALESCE(ANY_VALUE(tc.blob_link), "") blob_link,
	COALESCE(ANY_VALUE(tc.total_coverage), JSON_OBJECT()) total_coverage,
	COUNT(te.test_id),
	COUNT(CASE WHEN te.status = 'failed' THEN te.id END) failed,
	COUNT(CASE WHEN te.status = 'aborted' THEN te.id END) aborted,
	COUNT(CASE WHEN te.status = 'passed' THEN te.id END) passed,
	COUNT(CASE WHEN te.status = 'blocklisted' THEN te.id END) blocklisted,
	COUNT(CASE WHEN te.status = 'quarantined' THEN te.test_id END) quarantined,
	COUNT(CASE WHEN te.status = 'skipped' THEN te.id END) skipped
FROM
	git_commits gc
JOIN build b ON
	b.commit_id = gc.commit_id
LEFT JOIN test_execution te ON
	te.build_id = b.id
JOIN repositories r ON
	r.id = gc.repo_id
LEFT JOIN test_coverage tc ON
	tc.commit_id = gc.commit_id
	AND tc.repo_id = r.id
JOIN branch_commit bc ON
	bc.commit_id = gc.commit_id 
	AND bc.repo_id = r.id 
WHERE
	r.name = ?
	AND r.org_id = ?
	AND bc.branch_name = ?
	AND b.status IN ('passed', 'failed', 'error')
GROUP BY
	gc.commit_id
ORDER BY
	commit_latest_creation_date DESC
LIMIT ?
`

const findAuthorDataQuery = `
WITH authors_test AS (
	SELECT
			t.id test_id,
			gc.author_name
	FROM
			test t
	RIGHT JOIN git_commits gc ON
		t.debut_commit = gc.commit_id
		AND t.repo_id = gc.repo_id
	JOIN repositories r ON
			t.repo_id = r.id
		%s
	WHERE
			r.name = :repo
		AND r.org_id = :org_id
		%s
	),
	test_latest_status_ranked AS (
	SELECT
			t.author_name,
			t.test_id test_id,
			te.status,
			ROW_NUMBER() OVER (PARTITION BY te.test_id
	ORDER BY
		te.updated_at DESC ) rn
	FROM 
			test_execution te
	JOIN authors_test t ON
			t.test_id = te.test_id
	JOIN build b ON
	 	b.id = te.build_id
	WHERE 
		b.id != ""
		%s
	),
	test_latest_status AS (
	SELECT 
			*
	FROM
			test_latest_status_ranked tlsr
	WHERE
			rn = 1
	),
	latest_builds_ranked AS (
	SELECT
	    b.build_num,
		b.id,
		b.tag,
		b.branch_name,
		b.status,
		gc.author_name,
		ROW_NUMBER() OVER (PARTITION BY gc.author_name 
		ORDER BY
			b.updated_at DESC ) rn
	FROM
		git_commits gc
	JOIN build b ON
		b.commit_id = gc.commit_id
	JOIN repositories r ON
		r.id = b.repo_id
	WHERE
		r.name = :repo
		AND r.org_id = :org_id
		%s 
	),
	latest_builds AS (
		SELECT
			*
		FROM
			latest_builds_ranked
		WHERE
			rn = 1
	),
	author_date AS (
		SELECT
			gc.author_name,
			MIN(gc.authored_date) authored_date
		FROM
			git_commits gc
		JOIN repositories r ON
				gc.repo_id = r.id
			%s
		WHERE
				r.name = :repo
			AND r.org_id = :org_id
			%s
		GROUP BY
			gc.author_name
	)
	SELECT
		COUNT(DISTINCT tls.test_id),
		COUNT(DISTINCT CASE WHEN tls.status = 'passed' THEN tls.test_id END) passed,
		COUNT(DISTINCT CASE WHEN tls.status = 'failed' THEN tls.test_id END) failed,
		COUNT(DISTINCT CASE WHEN tls.status = 'skipped' THEN tls.test_id END) skipped,
		COUNT(DISTINCT CASE WHEN tls.status = 'blocklisted' THEN tls.test_id END) blocklisted,
		COUNT(DISTINCT CASE WHEN tls.status = 'quarantined' THEN tls.test_id END) quarantined,
		COUNT(DISTINCT CASE WHEN tls.status = 'aborted' THEN tls.test_id END) aborted,
		ad.author_name,
		IFNULL(ANY_VALUE(ad.authored_date), TIMESTAMP("0001-01-01")),
		IFNULL(ANY_VALUE(lbs.build_num),0) latest_build_num,
		IFNULL(ANY_VALUE(lbs.id),"") latest_build_id,
		IFNULL(ANY_VALUE(lbs.tag),"") latest_build_tag,
		IFNULL(ANY_VALUE(lbs.branch_name),"") latest_build_branch,
		IFNULL(ANY_VALUE(lbs.status),"") latest_build_status
	FROM
		author_date ad
	LEFT JOIN latest_builds lbs ON
		ad.author_name = lbs.author_name
	LEFT JOIN test_latest_status tls ON
		ad.author_name = tls.author_name
`
const findCommitMetaQuery = `
WITH commits_latest_build AS (
	SELECT
		*
	FROM (
		SELECT
			b.id,
			b.status,
			b.commit_id,
			ROW_NUMBER() OVER (PARTITION BY b.commit_id ORDER BY b.build_num DESC) rn
		FROM
			build b
		JOIN repositories r ON
			r.id = b.repo_id
		WHERE
			r.name = :repo
			AND r.org_id = :org_id
			%s
	) ranked_commit_builds
	WHERE
		rn = 1
)
SELECT
	COALESCE(COUNT(DISTINCT gc.commit_id),0) total,
	COALESCE(COUNT(DISTINCT CASE WHEN clb.status = 'passed' THEN gc.commit_id END),0) passed,
	COALESCE(COUNT(DISTINCT CASE WHEN clb.status = 'failed' THEN gc.commit_id END),0) failed,
	COALESCE(COUNT(DISTINCT CASE WHEN clb.status = 'skipped' THEN gc.commit_id END),0) skipped,
	COALESCE(COUNT(DISTINCT CASE WHEN clb.status = 'error' THEN gc.commit_id END),0) error,
	COALESCE(COUNT(DISTINCT CASE WHEN clb.status = 'aborted' THEN gc.commit_id END),0) aborted,
	COALESCE(COUNT(DISTINCT CASE WHEN clb.status = 'initiating' THEN gc.commit_id END),0) initiating,
	COALESCE(COUNT(DISTINCT CASE WHEN clb.status = 'running' THEN gc.commit_id END),0) running
	FROM
		git_commits gc 
	LEFT JOIN commits_latest_build clb ON
		clb.commit_id = gc.commit_id 
	JOIN repositories r ON
		r.id = gc.repo_id 
		%s
	WHERE
		r.name = :repo
		AND r.org_id = :org_id
		%s
		
`
const findContributorGraphQuery = `
SELECT
	gc.author_name AS author_name,
	JSON_ARRAYAGG(DATEDIFF(gc.authored_date, NOW() + INTERVAL -1 MONTH)) AS graph
FROM
	branch_commit bc
JOIN git_commits gc ON
	bc.commit_id = gc.commit_id
JOIN repositories r ON
	gc.repo_id = r.id
WHERE
	r.name = :repo
	AND r.org_id = :org_id
	%s
GROUP BY
	gc.author_name
ORDER BY author_name DESC
`

const findAuthorStatsQuery = `
WITH recent_author_commit AS (
	SELECT 
		gc.author_name, 
		gc.commit_id AS latest_commit_id
	FROM
		branch_commit bc
	JOIN git_commits gc ON
		bc.commit_id = gc.commit_id
	JOIN repositories r ON
		gc.repo_id = r.id
	WHERE
		r.org_id = :org
		AND r.name = :repo
		AND gc.author_name = :author
		%s
	ORDER BY gc.committed_date DESC LIMIT 1),
flaky_tests AS (
	SELECT
		gc.author_name,
		COUNT(DISTINCT(te.test_id)) AS flaky_count
	FROM
		flaky_test_execution fte
	JOIN
		test_execution te ON
		fte.test_id = te.test_id
	JOIN branch_commit bc ON
		te.commit_id = bc.commit_id
	JOIN 
		git_commits gc ON
		bc.commit_id = gc.commit_id
	JOIN
		repositories r ON
		r.id = gc.repo_id
	WHERE
		fte.status = 'flaky'
		AND r.org_id = :org
		AND r.name = :repo
		AND gc.author_name = :author
		%s
	GROUP BY gc.author_name),
total_commits AS(
	SELECT
		gc.author_name,
		COUNT(DISTINCT gc.commit_id) AS commit_count
	FROM
		branch_commit bc
	JOIN git_commits gc ON
		bc.commit_id = gc.commit_id
	JOIN
		repositories r ON
		r.id = gc.repo_id
	WHERE
		r.org_id = :org
		AND r.name = :repo
		AND gc.author_name = :author
		AND gc.created_at >= :start_date
		AND gc.created_at <= :end_date
		%s
	GROUP BY gc.author_name
)
SELECT
	COALESCE(rac.author_name, "") AS author_name,
	COALESCE(rac.latest_commit_id, "") AS latest_commit_id,
	COALESCE(ft.flaky_count, 0) AS flaky_count,
	COALESCE(tc.commit_count, 0) AS commit_count
FROM
	recent_author_commit rac 
LEFT JOIN flaky_tests ft ON
	ft.author_name = rac.author_name
JOIN total_commits tc ON
	tc.author_name = rac.author_name
`
const findTransitionQuery = `
SELECT
	COALESCE(JSON_ARRAYAGG(te.status), JSON_ARRAY())
FROM
	test_execution te
JOIN branch_commit bc ON 
	te.commit_id = bc.commit_id
JOIN git_commits gc ON
	bc.commit_id = gc.commit_id
JOIN repositories r ON
	gc.repo_id = r.id
WHERE
	r.org_id = :org_id
	AND r.name = :repo
	AND gc.author_name = :author
	AND gc.created_at >= :start_date
	AND gc.created_at <= :end_date
	%s
GROUP BY
	te.test_id
ORDER BY
	MIN(gc.created_at)
`
