package core

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"gopkg.in/guregu/null.v4/zero"
)

// GitCommit represents a git commit.
type GitCommit struct {
	ID             string                  `json:"id,omitempty" db:"id"`
	CommitID       string                  `json:"commit_id" db:"commit_id"`
	ParentCommitID string                  `json:"-" db:"parent_commit_id"`
	Message        string                  `json:"message" db:"message"`
	RepoID         string                  `json:"-" db:"repo_id"`
	Created        time.Time               `json:"created_at" db:"created_at"`
	Updated        time.Time               `json:"-" db:"updated_at"`
	Link           string                  `json:"link" db:"link"`
	Author         Author                  `json:"author" db:"author"`
	Committer      Committer               `json:"committer" db:"committer"`
	TestsAdded     int                     `json:"tests_added"`
	LatestBuild    *Build                  `json:"latest_build,omitempty"`
	Status         string                  `json:"status,omitempty"`
	Coverage       *TestCoverage           `json:"coverage,omitempty"`
	TestMeta       *ExecutionMeta          `json:"execution_meta,omitempty"`
	FlakyMeta      *FlakyExecutionMetadata `json:"flaky_execution_meta,omitempty"`
	Meta           *TaskMeta               `json:"task_meta,omitempty"`
	BuildID        string                  `json:"build_id,omitempty"`
	BuildNum       int                     `json:"build_num,omitempty"`
	BuildTag       string                  `json:"build_tag,omitempty"`
	Graph          []int                   `json:"contributor_graph,omitempty"`
	Branch         string                  `json:"branch,omitempty"`
	CommitDiffURL  string                  `json:"commit_diff_url,omitempty"`
}

// AuthorMeta contains meta data for a git author
// type AuthorMeta struct {
// 	TotalCommits int       `json:"commits_count"`
// 	Date         time.Time `json:"commit_date"`
// }

// Author identifies a git commit author.
type Author struct {
	Name  string    `db:"name"`
	Email string    `db:"email"`
	Date  time.Time `db:"date"`
	// Login  string
	// Avatar string
}

// Committer identifies a git commit commiter.
type Committer struct {
	Name  zero.String `db:"name"`
	Email zero.String `db:"email"`
	Date  zero.Time   `db:"date"`
}

// AuthorStats contains stats of the author for the repo
type AuthorStats struct {
	Name              string  `json:"name" db:"author_name"`
	LatestCommit      string  `json:"latest_commit_id" db:"latest_commit_id"`
	FlakyCount        int     `json:"flaky_count" db:"flaky_count"`
	CommitCount       int     `json:"commit_count" db:"commit_count"`
	AverageTransition float64 `json:"transition"`
	BlocklistedTests  int     `json:"test_blocklisted"`
}

// GitCommitStore defines datastore operation for working with git_commits.
type GitCommitStore interface {
	// CreateInTx persists a new commit to the datastore and executes the statement within supplied transaction.
	CreateInTx(ctx context.Context, tx *sqlx.Tx, commits []*GitCommit) error
	// FindByCommitID checks if commit exists in the datastore.
	FindByCommitID(ctx context.Context, commitID, repoID string) (string, error)
	// FindByCommitIDTx checks if commit exists in the datastore in transaction.
	FindByCommitIDTx(ctx context.Context, tx *sqlx.Tx, commitID, repoID string) (string, error)
	// FindByRepo returns all the commits for a repo.
	FindByRepo(ctx context.Context, repoName, orgID, commitID, buildID,
		branchName, orgName, gitProvider, statusFilter, searchText string, authorsNames []string,
		offset, limit int) ([]*GitCommit, error)
	// FindImpactedTests returns all the  tests impacted by the commit.
	FindImpactedTests(ctx context.Context, commitID, buildID, taskID, repoName, orgID, statusFilter, searchText, branchName string,
		offset, limit int) ([]*TestExecution, error)
	// FindAuthors returns the author of each commit of the repository.
	FindAuthors(ctx context.Context, repoName, orgID, branchName, author, nextAuthor string, limit int) ([]*GitCommit, error)
	// FindAuthorCommitActivity returns the commit author activity of the repository.
	FindAuthorCommitActivity(ctx context.Context, repoName, orgID, nextAuthor string, limit int) (map[string]map[int]int, string, error)
	// FindByBuild returns all the commits that are there for a particular build ID.
	FindByBuild(ctx context.Context, repoName, orgID, buildID string, offset, limit int) ([]*GitCommit, error)
	// FindCommitStatus returns the status of latest `limit` commits.
	FindCommitStatus(ctx context.Context, repoName, orgID, branchName string, limit int) ([]*GitCommit, error)
	// FindCommitMeta returns the commit meta of all the commits
	FindCommitMeta(ctx context.Context, repoName, orgID, branchName string) ([]*TaskMeta, error)
	// FindContributoGraph returns the commit meta of all the commits
	FindContributorGraph(ctx context.Context, repoName, orgID, branchName string) ([]*GitCommit, error)
	// FindAuthorStats returns the stats of the author in the contributors section
	FindAuthorStats(ctx context.Context, repoName, orgID, branchName, author string, startDate, endDate time.Time) (*AuthorStats, error)
	// FindTransitions finds the number of transitions of the tests in a particular duration
	FindTransitions(ctx context.Context, repoName, orgID, branchName, author string, startDate, endDate time.Time) (int, error)
}

// CommitService provides access to the commit history from
// the external source code management service (e.g. GitHub).
type CommitService interface {
	// Find returns the commit information by sha.
	Find(ctx context.Context, scmProvider SCMDriver, oauth *Token, repo, sha string) (*GitCommit, error)
}
