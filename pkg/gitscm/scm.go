package gitscm

import (
	"net/http"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/drone/go-scm/scm"

	"github.com/drone/go-scm/scm/driver/bitbucket"
	"github.com/drone/go-scm/scm/driver/github"
	"github.com/drone/go-scm/scm/driver/gitlab"
	"github.com/drone/go-scm/scm/transport/oauth2"
)

// gitClientProvider provides the git scm client
type gitClientProvider struct {
	logger          lumber.Logger
	gitHubClient    *core.SCM
	gitLabClient    *core.SCM
	bitbucketClient *core.SCM
	// selfgitHubClient *core.SCM
	// selfgitLabClient *core.SCM
}

// New initializes GitClientProvider
func New(logger lumber.Logger) core.SCMProvider {
	return &gitClientProvider{
		logger:          logger,
		gitHubClient:    &core.SCM{Client: provideGithubClient(), Name: core.DriverGithub.String()},
		gitLabClient:    &core.SCM{Client: provideGitlabClient(), Name: core.DriverGitlab.String()},
		bitbucketClient: &core.SCM{Client: provideBitbucketClient(), Name: core.DriverBitbucket.String()},
	}
}

func (g *gitClientProvider) GetClient(scmClientID core.SCMDriver) (*core.SCM, error) {
	switch scmClientID {
	case core.DriverGithub:
		return g.gitHubClient, nil
	case core.DriverGitlab:
		return g.gitLabClient, nil
	case core.DriverBitbucket:
		return g.bitbucketClient, nil
	default:
		return nil, errs.ErrInvalidDriver
	}
}

func provideGithubClient() *scm.Client {
	client := github.NewDefault()
	client.Client = &http.Client{
		Transport: &oauth2.Transport{
			Source: oauth2.ContextTokenSource(),
			Base:   http.DefaultTransport,
		},
	}
	return client
}

func provideGitlabClient() *scm.Client {
	client := gitlab.NewDefault()
	client.Client = &http.Client{
		Transport: &oauth2.Transport{
			Source: oauth2.ContextTokenSource(),
			Base:   http.DefaultTransport,
		},
	}
	return client
}

func provideBitbucketClient() *scm.Client {
	client := bitbucket.NewDefault()
	client.Client = &http.Client{
		Transport: &oauth2.Transport{
			Source: oauth2.ContextTokenSource(),
			Base:   http.DefaultTransport,
		},
	}
	return client
}
