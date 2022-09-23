package core

import (
	"context"
)

// EventHeader is the kafka header for webhook event type.
const EventHeader = "event_type"

// EventType represents the webhook event
type EventType string

const (
	// EventPush represents the push event.
	EventPush EventType = "push"
	// EventPullRequest represents the pull request event.
	EventPullRequest EventType = "pull-request"
	// EventPing represents the ping event.
	EventPing EventType = "ping"
)

// ParserResponse response after extracting details from webhook
type ParserResponse struct {
	Repository            *Repository
	EventBlob             *EventBlob
	GitCommits            []*GitCommit
	License               *License
	RunnerType            Runner
	SecretPath            string
	TokenPath             string
	InstallationTokenPath string
	Rebuild               bool
	BuildTag              BuildTag
}

// HookParser parses the webhook from the source
// code management system.
type HookParser interface {
	// CreateAndScheduleBuild parses the webhook and creates a build and queues the build/
	CreateAndScheduleBuild(eventType EventType, driver SCMDriver, repoID string, msg []byte) error
}

// HookService manages post-commit hooks in the external
// source code management service (e.g. GitHub).
type HookService interface {
	// Create a new git scm webhook
	Create(ctx context.Context, token *Token, gitProvider SCMDriver, repo *Repository) error
	// Delete a git scm webhook
	Delete(ctx context.Context, user *GitUser, repo *Repository, token *Token) error
}

// HookStore is wrapper for executing all the queries related to a webhook event in a transaction.
type HookStore interface {
	// CreateEntities persists the webhook related data in the datastore and
	// check if build can be created
	CreateEntities(ctx context.Context, payload *ParserResponse) (createBuild bool, err error)
	// CreateBuild creates a new build  and tasks in the datastore.
	CreateBuild(ctx context.Context, payloadAddress string, payload *ParserResponse) (*Job, error)
}
