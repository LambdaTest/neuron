package core

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"
	"gopkg.in/guregu/null.v4"
)

// GitEvent represents the git webhook event.
type GitEvent struct {
	ID                string          `db:"id"`
	RepoID            string          `db:"repo_id"`
	EventName         EventType       `db:"event_name"`
	ChangeList        null.String     `db:"change_list"`
	CommitID          string          `db:"commit_id"`
	GitProviderHandle SCMDriver       `db:"git_provider_handle"`
	EventPayload      json.RawMessage `db:"event_payload"`
	Response          null.String     `db:"response"`
	StatusCode        null.Int        `db:"status_code"`
	Created           time.Time       `db:"created_at"`
	Updated           time.Time       `db:"updated_at"`
}

// CommitDiff contains link for comparison
type CommitDiff struct {
	Link string
}

// Payloadreponse contains information related to diff
type Payloadresponse struct {
	PullRequests *CommitDiff `json:"PullRequest"`
	Commit       *CommitDiff `json:"Commit"`
}

// GitEventStore defines datastore operation for working with git_events
type GitEventStore interface {
	// Create persists a new git event to the datastore and executes the statement within supplied transaction.
	CreateInTx(ctx context.Context, tx *sqlx.Tx, event *GitEvent) error
	// FindByBuildID returns git event for a build.
	FindByBuildID(ctx context.Context, buildID string) (*GitEvent, error)
}
