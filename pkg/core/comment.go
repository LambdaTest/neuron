package core

import (
	"context"
)

// CommentService Create comments on issues/PRs
type CommentService interface {
	// CreateIssueComment creates a comment on issue/PR.
	CreateIssueComment(ctx context.Context, gitProvider SCMDriver,
		issueNumber int, tokenPath, installationTokenPath, repoSlug, body string) error
	// CreateFlakyComment Creates a comment with ftm results on issue/PR
	CreateFlakyComment(ctx context.Context, buildCache *BuildCache,
		flakyMeta *FlakyExecutionMetadata, buildID string)
}
