package core

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

// UserOrg represents a git user's organizations
type UserOrg struct {
	ID      string    `db:"id"`
	UserID  string    `db:"user_id"`
	OrgID   string    `db:"org_id"`
	Created time.Time `db:"created_at"`
	Updated time.Time `db:"updated_at"`
}

// UserOrgStore defines operations for working with user_organization store.
type UserOrgStore interface {
	// Create persists a new userorg in the datastore and executes the statement within the transaction.
	CreateInTx(ctx context.Context, tx *sqlx.Tx, userOrgs []*UserOrg) error
	// FindIfExists finds if the user org relation exists.
	FindIfExists(ctx context.Context, userID, orgID string) error
}
