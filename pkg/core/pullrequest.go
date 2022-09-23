package core

import (
	"context"

	"github.com/drone/go-scm/scm"
)

// PullRequestService provides pullrequest object
type PullRequestService interface {
	// Find returns pullrequest object by prNumber.
	Find(ctx context.Context, scmProvider SCMDriver, oauth *Token, reposlug string, prNumber int) (*scm.PullRequest, error)
}
