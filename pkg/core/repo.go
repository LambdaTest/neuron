package core

import (
	"context"
	"time"
)

// Repository represents a git repository

type JobView string

const (
	DefaultView JobView = "default"
	FtmOnlyView JobView = "ftm_only"
)

type Repository struct {
	ID                string    `json:"id,omitempty" db:"id"`
	OrgID             string    `json:"-" db:"org_id"`
	Strict            bool      `json:"-" db:"strict"`
	Name              string    `json:"name,omitempty" db:"name"`
	Admin             string    `json:"-" db:"admin_id"`
	Namespace         string    `json:"namespace,omitempty"`
	Private           bool      `json:"private" db:"private"`
	Secret            string    `json:"-" db:"webhook_secret"`
	Link              string    `json:"link" db:"link"`
	HTTPURL           string    `json:"http_url,omitempty" db:"git_http_url"`
	SSHURL            string    `json:"ssh_url,omitempty" db:"git_ssh_url"`
	Active            bool      `json:"active" db:"active"`
	TasFileName       string    `json:"-"  db:"tas_file_name"`
	PostMergeStrategy int       `json:"-"  db:"post_merge_strategy"`
	Created           time.Time `json:"created_at"  db:"created_at"`
	Updated           time.Time `json:"updated_at"  db:"updated_at"`
	Perm              *Perm     `json:"permissions,omitempty"`
	Mask              string    `json:"-" db:"mask"`
	CollectCoverage   bool      `json:"-" db:"collect_coverage"`
	Meta              *RepoMeta `json:"metadata,omitempty"`
	LatestBuild       *Build    `json:"latest_build,omitempty"`
	RepoGraph         []int     `json:"repo_graph,omitempty"`
	JobView           JobView   `json:"job_view" db:"job_view"`
}

// RepoMeta contains additional info of the builds
type RepoMeta struct {
	TotalTests int `json:"total_tests"`
	*BuildMeta
}

// Perm represents the user' s repository permissions
type Perm struct {
	Read  bool `db:"perm_read"     json:"read"`
	Write bool `db:"perm_write"    json:"write"`
	Admin bool `db:"perm_admin"    json:"admin"`
}

// Badge represents struct for badge data
type Badge struct {
	BuildStatus      string  `json:"status"`
	Duration         string  `json:"duration"`
	TotalTests       int     `json:"total_tests"`
	Passed           int     `json:"passed"`
	Failed           int     `json:"failed"`
	Skipped          int     `json:"skipped"`
	DiscoveredTests  int     `json:"discovered_tests"`
	PercentTimeSaved float64 `json:"time_saved_percent"`
	BuildID          string  `json:"build_id"`
	ExecutionTime    int     `json:"-"`
	TimeSaved        string  `json:"time_saved"`
}

// BadgeFlaky represents struct for badge related to flaky data
type BadgeFlaky struct {
	BuildStatus   string `json:"job_status"`
	BuildID       string `json:"job_id"`
	FlakyTests    int    `json:"flaky_tests"`
	NonFlakyTests int    `json:"non_flaky_tests"`
	Quarantined   int    `json:"quarantined_tests"`
}

// TokenPathInfo represents Info required to create git token path
type TokenPathInfo struct {
	OrgID       string    `db:"org_id"`
	OrgName     string    `db:"org_name"`
	UserID      string    `db:"admin_id"`
	UserMask    string    `db:"user_mask"`
	GitProvider SCMDriver `db:"git_provider"`
}

// RepoSettingsInfo is the request body for the repo settings
type RepoSettingsInfo struct {
	OrgName        string  `json:"org,omitempty"`
	RepoName       string  `json:"repo,omitempty"`
	Strict         bool    `json:"strict"`
	ConfigFileName string  `json:"config_file_name"`
	JobView        JobView `json:"job_view"`
}

// RepoStore defines operations for working with repositories.
type RepoStore interface {
	// Create persists a new repository in the data store.
	Create(ctx context.Context, repo *Repository) error
	// Find returns the repository in the data store.
	Find(ctx context.Context, orgID string, name string) (*Repository, error)
	// Find returns the repository in the data store by repoID.
	FindByID(ctx context.Context, repoID string) (*Repository, error)
	// FindByCommitID returns the repository in the data store by commitID.
	FindByCommitID(ctx context.Context, commitID string) (repoName string, authorName string, err error)
	// FindActiveByName returns the active repositories in the data store by commitID.
	FindActiveByName(ctx context.Context, repoName, orgName string, gitProvider SCMDriver) (repo *Repository, err error)
	// FindIfActive returns the repository if active.
	FindIfActive(ctx context.Context, orgID string, name string) (*Repository, error)
	// FindAllActiveMap returns all the active repositories for an org in a map.
	FindAllActiveMap(ctx context.Context, orgID string) (map[string]struct{}, error)
	// FindAllActive returns all the active repositories for a user in a slice.
	FindAllActive(ctx context.Context, orgID, searchText string, offset, limit int) ([]*Repository, error)
	// UpdateStatus updates the status of info in repo table related to config files.
	UpdateStatus(ctx context.Context, repoConfig RepoSettingsInfo, orgID, userID string) error
	// FindByBuildID returns repo by buildID
	FindByBuildID(ctx context.Context, buildID string) (*Repository, error)
	// FindBadgeData returns the data of badge
	FindBadgeData(ctx context.Context, repoName, orgName, branchName, buildID, gitProvider string) (*Badge, error)
	// RemoveRepo deactivate the repo from repo import page
	RemoveRepo(ctx context.Context, repoName, orgID, userID string) error
	// FindTokenPathInfo return the info required to create token path to vault
	FindTokenPathInfo(ctx context.Context, repoID string) (*TokenPathInfo, error)
	// FindBadgeDataFlaky returns the data of badge related to flaky jobs
	FindBadgeDataFlaky(ctx context.Context, repoName, orgName, branchName, gitProvider string) (*BadgeFlaky, error)
}

// RepositoryService provides access to repository information
// in the remote source code management system (e.g. GitHub).
type RepositoryService interface {
	// Find returns a repository by name.
	Find(ctx context.Context, repoSlug string, token *Token, gitProvider SCMDriver) (*Repository, error)
	// List returns a list of repositories.
	List(ctx context.Context, page int, size int, userID, orgID, orgName string, user *GitUser) ([]*Repository, int, error)
	// ListBranches returns a list of branches of a repo.
	ListBranches(ctx context.Context, page, size int, repoName string, user *GitUser) (branches []string, next int, err error)
}
