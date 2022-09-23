package hook

import (
	"context"
	"fmt"
	"net/url"

	"github.com/LambdaTest/neuron/config"
	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/drone/go-scm/scm"
)

type service struct {
	scmProvider         core.SCMProvider
	hookName            string
	addr                string
	logger              lumber.Logger
	skipSSLVerification bool
}

// New returns a new HookService.
func New(scmProvider core.SCMProvider, cfg *config.Config, logger lumber.Logger) core.HookService {
	if cfg.WebhookAddress == "" {
		logger.Fatalf("webhook address not found")
	}

	hookName, ok := constants.GitStatusLabel[cfg.Env]
	if !ok {
		logger.Fatalf("Invalid Environment %v", cfg.Env)
	}
	skipSSLVerification := false
	if cfg.Env == constants.Dev {
		skipSSLVerification = true
	}
	u, err := url.Parse(cfg.WebhookAddress)
	if err != nil {
		logger.Fatalf("Invalid Hook Address url %s, error %v", cfg.WebhookAddress, err)
	}

	return &service{scmProvider: scmProvider,
		hookName:            hookName,
		addr:                u.String(),
		logger:              logger,
		skipSSLVerification: skipSSLVerification}
}

func (s *service) Create(ctx context.Context, token *core.Token, gitProvider core.SCMDriver, repo *core.Repository) error {
	gitSCM, err := s.scmProvider.GetClient(gitProvider)
	if err != nil {
		s.logger.Errorf("failed to find git client for driver: %s, error: %v", gitProvider, err)
		return err
	}
	ctx = token.SetInstallationRequestContext(ctx)

	hookInput := &scm.HookInput{
		Name:       s.hookName,
		Target:     fmt.Sprintf("%s/hook/%s", s.addr, gitProvider),
		SkipVerify: s.skipSSLVerification,
		Secret:     repo.Secret,
		Events: scm.HookEvents{
			PullRequest: true,
			Push:        true,
		},
	}

	if err := s.replaceHook(ctx, gitSCM.Client, scm.Join(repo.Namespace, repo.Name), hookInput); err != nil {
		s.logger.Errorf("failed to create hook %v", err)
		return err
	}
	return nil
}

func (s *service) Delete(ctx context.Context, user *core.GitUser, repo *core.Repository, token *core.Token) error {
	gitSCM, err := s.scmProvider.GetClient(user.GitProvider)
	if err != nil {
		s.logger.Errorf("failed to find git client for driver: %s, error: %v", user.GitProvider, err)
		return err
	}

	ctx = token.SetInstallationRequestContext(ctx)

	return s.deleteHook(ctx, gitSCM.Client, scm.Join(repo.Namespace, repo.Name), s.addr)
}

func (s *service) replaceHook(ctx context.Context, client *scm.Client, repo string, hook *scm.HookInput) error {
	if err := s.deleteHook(ctx, client, repo, hook.Target); err != nil {
		s.logger.Debugf("failed to delete hook for repo %v, error %v", repo, err)
		return err
	}
	h, resp, err := client.Repositories.CreateHook(ctx, repo, hook)
	if err != nil {
		s.logger.Debugf("failed to create hook for repo %v, resp %+v, error %v", repo, resp, err)
		return err
	}
	s.logger.Debugf("created hook with target %s, id %s", hook.Target, h.ID)
	return nil
}

func (s *service) deleteHook(ctx context.Context, client *scm.Client, repo, target string) error {
	u, err := url.Parse(target)
	if err != nil {
		s.logger.Debugf("failed to parse target for repo %v error %v", repo, err)
		return err
	}
	h, err := s.findHook(ctx, client, repo, u.Host)
	if err != nil {
		s.logger.Debugf("failed to find hook for repo %v err %v", repo, err)
		return err
	}
	if h == nil {
		return nil
	}
	_, err = client.Repositories.DeleteHook(ctx, repo, h.ID)
	s.logger.Debugf("Delete hook scm for repo %v error %v", repo, err)
	return err
}

func (s *service) findHook(ctx context.Context, client *scm.Client, repo, host string) (*scm.Hook, error) {
	hooks, _, err := client.Repositories.ListHooks(ctx, repo, scm.ListOptions{Size: 100})
	if err != nil {
		return nil, err
	}
	for _, hook := range hooks {
		u, err := url.Parse(hook.Target)
		if err != nil {
			continue
		}
		if u.Host == host {
			return hook, nil
		}
	}
	return nil, nil
}
