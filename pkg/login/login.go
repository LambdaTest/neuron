package login

import (
	"net/http"
	"strings"

	"github.com/LambdaTest/neuron/config"
	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/drone/go-login/login"
	"github.com/drone/go-login/login/bitbucket"
	"github.com/drone/go-login/login/github"
	"github.com/drone/go-login/login/gitlab"
)

// gitLoginProvider  that returns a git authenticator middleware
type gitLoginProvider struct {
	logger         lumber.Logger
	githublogin    login.Middleware
	gitlablogin    login.Middleware
	bitbucketlogin login.Middleware
}

// New returns new GitLoginProvider
func New(cfg *config.Config, logger lumber.Logger) core.GitLoginProvider {
	return &gitLoginProvider{
		logger:         logger,
		githublogin:    provideGithubLogin(cfg, logger),
		gitlablogin:    provideGitlabLogin(cfg, logger),
		bitbucketlogin: provideBitbucketLogin(cfg, logger),
	}
}

// Get returns the login middleware
func (g *gitLoginProvider) Get(clientID core.SCMDriver) (login.Middleware, error) {
	switch clientID {
	case core.DriverGithub:
		return g.githublogin, nil
	case core.DriverGitlab:
		return g.gitlablogin, nil
	case core.DriverBitbucket:
		return g.bitbucketlogin, nil

	default:
		return nil, errs.ErrInvalidDriver
	}
}

// provideGithubLogin returns a GitHub authenticator
// based on the environment configuration
func provideGithubLogin(cfg *config.Config, logger lumber.Logger) login.Middleware {
	if cfg.GitHub.ClientID == "" {
		logger.Fatalf("missing GitHub ClientID")
	}
	if cfg.GitHub.ClientSecret == "" {
		logger.Fatalf("missing GitHub ClientSecret")
	}
	if cfg.GitHub.Scope == "" {
		logger.Fatalf("missing GitHub PrivateRepoScope")
	}

	//TODO: go-login logger
	return &github.Config{
		ClientID:     cfg.GitHub.ClientID,
		ClientSecret: cfg.GitHub.ClientSecret,
		Scope:        strings.Split(cfg.GitHub.Scope, ","),
		Server:       cfg.GitHub.Server,
		Client:       &http.Client{},
	}
}

// provideGitlabLogin returns a Gitlab authenticator
// based on the environment configuration
func provideGitlabLogin(cfg *config.Config, logger lumber.Logger) login.Middleware {
	if cfg.GitLab.ClientID == "" {
		logger.Fatalf("missing GitLab ClientID")
	}
	if cfg.GitLab.ClientSecret == "" {
		logger.Fatalf("missing GitLab ClientSecret")
	}
	if cfg.GitLab.PrivateRepoScope == "" {
		logger.Fatalf("missing GitLab PrivateRepoScope")
	}

	return &gitlab.Config{
		ClientID:     cfg.GitLab.ClientID,
		ClientSecret: cfg.GitLab.ClientSecret,
		RedirectURL:  cfg.GitLab.RedirectURL,
		Scope:        strings.Split(cfg.GitLab.PrivateRepoScope, ","),
		Server:       cfg.GitLab.Server,
		Client:       &http.Client{},
	}
}

// provideBitbucketLogin returns a Bitbucket authenticator
// based on the environment configuration
func provideBitbucketLogin(cfg *config.Config, logger lumber.Logger) login.Middleware {
	if cfg.Bitbucket.ClientID == "" {
		logger.Fatalf("missing Bitbucket ClientID")
	}
	if cfg.Bitbucket.ClientSecret == "" {
		logger.Fatalf("missing Bitbucket ClientSecret")
	}

	return &bitbucket.Config{
		ClientID:     cfg.Bitbucket.ClientID,
		ClientSecret: cfg.Bitbucket.ClientSecret,
		RedirectURL:  cfg.Bitbucket.RedirectURL,
		Client:       &http.Client{},
	}
}
