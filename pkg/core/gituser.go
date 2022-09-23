package core

import (
	"context"
	"time"

	"github.com/drone/go-login/login"
	"github.com/jmoiron/sqlx"
)

// GitUser represents the git scm user
type GitUser struct {
	ID          string    `db:"id"`
	GitProvider SCMDriver `db:"git_provider"`
	Username    string    `db:"username"`
	Avatar      string    `db:"avatar"`
	Email       string    `db:"email"`
	Created     time.Time `db:"created_at"`
	Updated     time.Time `db:"updated_at"`
	Mask        string    `db:"mask"`
	Oauth       Token
}

// AuthorMeta contains meta data for a git author
type AuthorMeta struct {
	TotalCommits      int    `db:"total_commits" json:"commits_count"`
	TotalBuilds       int    `db:"total_builds" json:"builds_count"`
	TotalRepositories int    `db:"total_repos" json:"repositories_count"`
	TotalTests        int    `db:"total_tests" json:"tests_count"`
	Name              string `db:"author" json:"author"`
}

// GitUserService provides access to user account
// resources in the remote system (e.g. GitHub,GitLab).
type GitUserService interface {
	// Find returns the authenticated user.
	Find(ctx context.Context, driver SCMDriver, loginToken *login.Token) (*GitUser, error)
}

// GitUserStore defines datastore operation for working with gitusers
type GitUserStore interface {
	// Create persists a new user to the datastore.
	Create(ctx context.Context, user *GitUser) error
	// CreateInTx persists a new user to the datastore and executes the statement within the transaction.
	CreateInTx(ctx context.Context, tx *sqlx.Tx, user *GitUser) error
	// Find returns a user from the datastore.
	Find(ctx context.Context, username string, gitProvider SCMDriver) (*GitUser, error)
	// FindByID returns a user from the datastore by userID.
	FindByID(ctx context.Context, userID string) (*GitUser, error)
	// FindByOrg returns a user from the datastore by organization.
	FindByOrg(ctx context.Context, userID string, orgName string) (*GitUser, string, error)
	// FindUserTasUsageInfo finds the total usage of tas for a user
	FindUserTasUsageInfo(ctx context.Context, orgID, userID string) ([]*AuthorMeta, error)
	// FindAdminByOrgNameAndGitProvider finds the admin user of org using orgName and GitProvider
	FindAdminByOrgNameAndGitProvider(ctx context.Context, orgName, gitProvider string) (*GitUser, error)
	// FindAdminByBuildID finds admin's name & email from git_users table of the repo by using buildID
	FindAdminByBuildID(ctx context.Context, buildID string) (*GitUser, error)
}
