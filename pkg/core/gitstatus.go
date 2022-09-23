package core

import "context"

// GitStatus is the git scm status which will be updated using APIs.
type GitStatus struct {
	Status   BuildStatus
	CommitID string
}

// GitStatusService sends the commit status to an external
// git SCM provider.
type GitStatusService interface {
	// UpdateGitStatus updates status on Git SCM provider.
	UpdateGitStatus(ctx context.Context, gitProvider SCMDriver, repoSlug, buildID, commitID, tokenPath, installationTokenPath string,
		status BuildStatus, buildTag BuildTag) error
}
