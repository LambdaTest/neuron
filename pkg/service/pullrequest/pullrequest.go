package pullrequest

import (
	"context"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/drone/go-scm/scm"
)

type service struct {
	scmProvider core.SCMProvider
	logger      lumber.Logger
}

// New returns a new PullRequestFactory.
func New(scmProvider core.SCMProvider, logger lumber.Logger) core.PullRequestService {
	return &service{
		scmProvider: scmProvider,
		logger:      logger,
	}
}

// Find returns the pullrequest information by reposlug and prNumber.
func (s *service) Find(ctx context.Context, scmProvider core.SCMDriver,
	oauth *core.Token, repoSlug string, prNumber int) (*scm.PullRequest, error) {
	gitSCM, err := s.scmProvider.GetClient(scmProvider)
	if err != nil {
		s.logger.Errorf("failed to find git client %v", err)
		return nil, err
	}
	// set token in context for the git scm client
	ctx = oauth.SetRequestContext(ctx)
	pullrequest, _, err := gitSCM.Client.PullRequests.Find(ctx, repoSlug, prNumber)
	if err != nil {
		return nil, err
	}
	return pullrequest, nil
}
