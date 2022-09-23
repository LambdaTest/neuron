package core

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

// CreditsUsage represents information on how the credits were consumed by user of an organization
type CreditsUsage struct {
	ID            string    `db:"id" json:"-"`
	OrgID         string    `db:"org_id" json:"org_id"`
	User          string    `db:"user" json:"user"`
	TaskID        string    `db:"task_id" json:"task_id"`
	Created       time.Time `db:"created_at" json:"-"`
	Updated       time.Time `db:"updated_at" json:"-"`
	Consumed      int32     `json:"credits_consumed"`
	Tier          Tier      `json:"tier"`
	Duration      float64   `json:"duration"`
	MachineConfig string    `json:"machine_config"`
	CommitID      string    `json:"commit_id,omitempty"`
	BuildID       string    `json:"build_id,omitempty"`
	RepoName      string    `json:"repo_name,omitempty"`
	BuildTag      string    `json:"build_tag,omitempty"`
}

// CreditsUsageStore defines datastore operation for working with credits_usage
type CreditsUsageStore interface {
	// CreateInTx persists the users credit usage in datastore and executes the statement in the specified transaction
	CreateInTx(ctx context.Context, tx *sqlx.Tx, usage *CreditsUsage) error
	// FindByOrgOrUser returns the credit usage from datastore by user or organization.
	FindByOrgOrUser(ctx context.Context, user, orgID string, startDate, endDate time.Time, offset, limit int) ([]*CreditsUsage, error)
	// FindTotalConsumedCredits returns the total consumed credit for an org with given timeline
	FindTotalConsumedCredits(ctx context.Context, orgID string, startDate, endDate time.Time) (int32, error)
}
