package commit

import (
	"context"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"gopkg.in/guregu/null.v4/zero"
)

type service struct {
	scmProvider core.SCMProvider
	logger      lumber.Logger
}

// New returns a new CommitServiceFactory.
func New(scmProvider core.SCMProvider, logger lumber.Logger) core.CommitService {
	return &service{
		scmProvider: scmProvider,
		logger:      logger,
	}
}

// Find returns the commit information by sha.
func (s *service) Find(ctx context.Context, scmProvider core.SCMDriver, oauth *core.Token, repo, sha string) (*core.GitCommit, error) {
	gitSCM, err := s.scmProvider.GetClient(scmProvider)
	if err != nil {
		s.logger.Errorf("failed to find git client %v", err)
		return nil, err
	}
	// set token in context for the git scm client
	ctx = oauth.SetRequestContext(ctx)

	commit, _, err := gitSCM.Client.Git.FindCommit(ctx, repo, sha)
	if err != nil {
		return nil, err
	}

	if scmProvider == core.DriverBitbucket {
		commit.Author.Name = commit.Author.Email
		commit.Committer.Name = commit.Committer.Email
	}

	return &core.GitCommit{
		CommitID: commit.Sha,
		Message:  commit.Message,
		Link:     commit.Link,
		Author: core.Author{
			Name:  utils.GetAuthorName(&commit.Author),
			Email: commit.Author.Email,
			Date:  commit.Author.Date,
		},
		Committer: core.Committer{
			Name:  zero.StringFrom(commit.Committer.Name),
			Email: zero.StringFrom(commit.Committer.Email),
			Date:  zero.TimeFrom(commit.Committer.Date),
		},
	}, nil
}
