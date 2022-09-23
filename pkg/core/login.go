package core

import (
	"context"

	"github.com/drone/go-login/login"
)

// GitLoginProvider returns a git authenticator middleware
type GitLoginProvider interface {
	// Get returns the login middleware based on the type of scm provider (eg. github)
	Get(clientID SCMDriver) (login.Middleware, error)
}

// LoginStore is wrapper for executing all the queries related to a webhook event in a transaction.
type LoginStore interface {
	// Create persists the data in the datastore.
	Create(ctx context.Context, user *GitUser, userExists bool, orgs ...*Organization) error
	// UpdateOrgList  updates the org list in db
	UpdateOrgList(ctx context.Context, user *GitUser, orgs []*Organization) ([]*Organization, error)
}
