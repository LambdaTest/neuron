package gitstatus

import (
	"context"
	"fmt"
	"strconv"

	"github.com/LambdaTest/neuron/config"
	"github.com/LambdaTest/neuron/pkg/constants"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/drone/go-scm/scm"
)

type service struct {
	scmProvider  core.SCMProvider
	tokenHandler core.GitTokenHandler
	repoStore    core.RepoStore
	logger       lumber.Logger
	targetURL    string
	defaultLabel string
}

// New returns a new Repository service, providing access to the
// repository information from the source code management system.
func New(cfg *config.Config,
	repoStore core.RepoStore,
	scmProvider core.SCMProvider,
	tokenHandler core.GitTokenHandler,
	logger lumber.Logger) core.GitStatusService {
	if cfg.FrontendURL == "" {
		logger.Fatalf("failed to find target url for env %s", cfg.Env)
	}
	defaultLabel, ok := constants.GitStatusLabel[cfg.Env]
	if !ok {
		logger.Fatalf("failed to find git status label for env %s", cfg.Env)
	}
	return &service{
		scmProvider:  scmProvider,
		tokenHandler: tokenHandler,
		repoStore:    repoStore,
		targetURL:    cfg.FrontendURL,
		logger:       logger,
		defaultLabel: defaultLabel,
	}
}

func (s *service) UpdateGitStatus(
	ctx context.Context,
	gitProvider core.SCMDriver,
	repoSlug, buildID, commitID, tokenPath, installationTokenPath string,
	status core.BuildStatus,
	buildTag core.BuildTag) error {

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

	input := &scm.StatusInput{
		Target: fmt.Sprintf("%s/%s/%s/jobs/%s/", s.targetURL, gitProvider.String(), repoSlug, buildID),
		Label:  s.defaultLabel,
		Desc:   createDesc(status),
		State:  createStatus(status),
	}

	orgName, repoName := scm.Split(repoSlug)
	if (status == core.BuildPassed || status == core.BuildFailed) && orgName != "" && repoName != "" {
		statusData, serr := s.repoStore.FindBadgeData(ctx, repoName, orgName, "", buildID, gitProvider.String())
		if serr != nil {
			s.logger.Errorf("failed to find test status for buildID: %s, error: %v", buildID, serr)
			return serr
		}

		if status == core.BuildPassed {
			if statusData.TotalTests == 1 {
				input.Desc = fmt.Sprintf("%s: %s test, %s.", input.Desc, strconv.Itoa(statusData.TotalTests), statusData.Duration)
			} else {
				input.Desc = fmt.Sprintf("%s: %s tests, %s.", input.Desc, strconv.Itoa(statusData.TotalTests), statusData.Duration)
			}
		} else {
			if statusData.Failed == 1 {
				input.Desc = fmt.Sprintf("%s: %s test failed.", input.Desc, strconv.Itoa(statusData.Failed))
			} else {
				input.Desc = fmt.Sprintf("%s: %s tests failed.", input.Desc, strconv.Itoa(statusData.Failed))
			}
		}
	}

	ctx = token.SetInstallationRequestContext(ctx)
	_, _, err = gitSCM.Client.Repositories.CreateStatus(ctx, repoSlug, commitID, input)

	return err
}
