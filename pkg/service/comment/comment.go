package comment

import (
	"context"

	"github.com/LambdaTest/neuron/config"
	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/drone/go-scm/scm"
)

type service struct {
	scmProvider  core.SCMProvider
	tokenHandler core.GitTokenHandler
	frontendURL  string
	logger       lumber.Logger
}

// New returns a new commment service, providing ability to
// commengt on a issue
func New(cfg *config.Config,
	scmProvider core.SCMProvider,
	tokenHandler core.GitTokenHandler,
	logger lumber.Logger) core.CommentService {
	return &service{
		scmProvider:  scmProvider,
		tokenHandler: tokenHandler,
		frontendURL:  cfg.FrontendURL,
		logger:       logger,
	}
}

func (s *service) CreateFlakyComment(ctx context.Context,
	buildCache *core.BuildCache,
	flakyMeta *core.FlakyExecutionMetadata, buildID string) {
	body := utils.GetFlakyComment(flakyMeta.FlakyTests, s.frontendURL, buildCache.GitProvider.String(), buildCache.RepoSlug, buildID)

	if err := s.CreateIssueComment(ctx,
		buildCache.GitProvider, buildCache.PullRequestNumber,
		buildCache.TokenPath, buildCache.InstallationTokenPath, buildCache.RepoSlug, body); err != nil {
		s.logger.Errorf("Failed to create comment on PR %s, repo %s, err %s", buildCache.PullRequestNumber, buildCache.RepoSlug, err)
	}
}

func (s *service) CreateIssueComment(
	ctx context.Context,
	gitProvider core.SCMDriver, issueNumber int, tokenPath, installationTokenPath,
	repoSlug, body string) error {
	if gitProvider != core.DriverGithub {
		return nil
	}

	token, _, err := s.tokenHandler.GetTokenFromPath(ctx, gitProvider, tokenPath, installationTokenPath)
	if err != nil {
		s.logger.Errorf("failed to get token for path %s, error: %v", tokenPath, err)
		return err
	}

	gitSCM, err := s.scmProvider.GetClient(gitProvider)
	if err != nil {
		s.logger.Errorf("failed to find git client for driver: %s, error: %v", gitProvider, err)
		return err
	}

	input := &scm.CommentInput{
		Body: body,
	}

	ctx = token.SetInstallationRequestContext(ctx)
	_, _, err = gitSCM.Client.Issues.CreateComment(ctx, repoSlug, issueNumber, input)

	return err
}
