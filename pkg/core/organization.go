package core

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"gopkg.in/guregu/null.v4/zero"
)

// Organization represents a git organization
type Organization struct {
	ID          string      `json:"id" db:"id"`
	Name        string      `json:"name" db:"name"`
	Avatar      string      `json:"avatar" db:"avatar"`
	GitProvider SCMDriver   `json:"git_provider" db:"git_provider"`
	Created     time.Time   `json:"-" db:"created_at"`
	Updated     time.Time   `json:"-"  db:"updated_at"`
	SecretKey   zero.String `json:"synapse_key" db:"secret_key"`
	RunnerType  Runner      `json:"runner_type" db:"runner_type"`
}

// Runner defines type of runner
type Runner string

// Possible Runner types
const (
	SelfHosted  Runner = "self-hosted"
	CloudRunner Runner = "cloud-runner"
)

// OrganizationService provides access to organization and
// team access in the external source code management system
// (e.g. GitHub).
type OrganizationService interface {
	// List returns a list of organization to which the
	// user is a member.
	List(context.Context, SCMDriver, *GitUser, *Token) ([]*Organization, error)
	// GetOrganization returns org details for an org name
	GetOrganization(ctx context.Context, driver SCMDriver, user *GitUser, t *Token, orgName string) (*Organization, error)
}

// OrganizationStore defines operations for working with organizations.
type OrganizationStore interface {
	// Create persists a new organization in datastore.
	Create(ctx context.Context, orgs []*Organization) error
	// CreateInTx persists a new organization in datastore and executes the statement within the transaction.
	CreateInTx(ctx context.Context, tx *sqlx.Tx, orgs []*Organization) error
	// FindOrCreate finds if organization exists otherwise creates a new organization.
	FindOrCreate(ctx context.Context, orgs *Organization) (string, error)
	// Find returns the organization in the datastore.
	Find(ctx context.Context, org *Organization) (string, error)
	// FindRunnerType returns the runner type of the organization in the datastore.
	FindRunnerType(ctx context.Context, orgID string) (Runner, error)
	// FindOrgsByNameTx finds the organizations in the datastore by name and executes the statement within the transaction.
	FindOrgsByNameTx(ctx context.Context, tx *sqlx.Tx, gitProvider SCMDriver, orgs []*Organization) ([]*Organization, error)
	// FindUserOrgs returns the user's organizations.
	FindUserOrgs(ctx context.Context, userID string) ([]*Organization, error)
	// FindUserOrgsTx returns the user's organizations and executes the statement within the transaction.
	FindUserOrgsTx(ctx context.Context, tx *sqlx.Tx, userID string) ([]*Organization, error)
	// FindByID returns the organization by the ID.
	FindByID(ctx context.Context, orgID string) (*Organization, error)
	// GetOrgCache returns the org concurrency details in cache.
	GetOrgCache(ctx context.Context, orgID string) (*OrgDetailsCache, error)
	// UpdateKeysInCache updates the org concurrency details in cache.
	UpdateKeysInCache(ctx context.Context, orgID string, concurrencyKeys map[string]int64) error
	// UpdateConfigInfo updates the org config info
	UpdateConfigInfo(ctx context.Context, orgID, secretKey, runnerType string) error
	// UpdateSynapseInfo updates the org synapse info
	UpdateSynapseToken(ctx context.Context, orgID, secretKey, token string) error
	// FindBySecretKey returns the organization by the secretKey.
	FindBySecretKey(ctx context.Context, secretKey string) (*Organization, error)
}

// OrgDetailsCache details related to a particular org, is cached in redis
type OrgDetailsCache struct {
	QueuedTasks      int    `redis:"queued_tasks"`
	RunningTasks     int    `redis:"running_tasks"`
	TotalConcurrency int    `redis:"total_concurrency"`
	RunnerType       string `redis:"runner_type"`
}
