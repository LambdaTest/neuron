package organizations

import (
	"context"
	"time"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/utils"

	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/drone/go-scm/scm"
)

// New returns a new OrganizationService.
func New(scmProvider core.SCMProvider, logger lumber.Logger) core.OrganizationService {
	return &service{scmProvider: scmProvider, logger: logger}
}

type service struct {
	scmProvider core.SCMProvider
	logger      lumber.Logger
}

func (s *service) List(ctx context.Context, driver core.SCMDriver, user *core.GitUser, t *core.Token) ([]*core.Organization, error) {
	gitSCM, err := s.scmProvider.GetClient(driver)
	if err != nil {
		s.logger.Errorf("failed to find git client  for driver: %s, error: %v", driver.String(), err)
		return nil, err
	}
	ctx = t.SetRequestContext(ctx)

	out, _, err := gitSCM.Client.Organizations.List(ctx, scm.ListOptions{Size: 100})
	if err != nil {
		s.logger.Errorf("failed to find organizations for the user: %s, provider: %s, error: %v", user.Username, user.GitProvider, err)
		return nil, err
	}
	now := time.Now()
	orgs := make([]*core.Organization, 0, len(out)+1)
	for _, org := range out {
		orgs = append(orgs, &core.Organization{
			ID:          utils.GenerateUUID(),
			Name:        org.Name,
			Avatar:      org.Avatar,
			GitProvider: user.GitProvider,
			Updated:     now,
			Created:     now,
		})
	}

	// create user also as an organization (Not for bitbucket)
	if user.GitProvider == core.DriverGitlab {
		orgs = append(orgs, &core.Organization{ID: utils.GenerateUUID(),
			Name:        user.Username,
			Avatar:      user.Avatar,
			GitProvider: user.GitProvider,
			Created:     now,
			Updated:     now})
	}

	return orgs, nil
}

func (s *service) GetOrganization(ctx context.Context,
	driver core.SCMDriver,
	user *core.GitUser,
	t *core.Token,
	orgName string) (*core.Organization, error) {
	now := time.Now()
	if orgName == user.Username {
		return &core.Organization{ID: utils.GenerateUUID(),
			Name:        user.Username,
			Avatar:      user.Avatar,
			GitProvider: user.GitProvider,
			Created:     now,
			Updated:     now}, nil
	}

	gitSCM, err := s.scmProvider.GetClient(driver)
	if err != nil {
		s.logger.Errorf("failed to find git client  for driver: %s, error: %v", driver.String(), err)
		return nil, err
	}
	ctx = t.SetRequestContext(ctx)

	org, _, err := gitSCM.Client.Organizations.Find(ctx, orgName)
	if err != nil {
		s.logger.Errorf("failed to find organizations for the user: %s, provider: %s, error: %v", user.Username, user.GitProvider, err)
		return nil, err
	}

	return &core.Organization{
		ID:          utils.GenerateUUID(),
		Name:        org.Name,
		Avatar:      org.Avatar,
		GitProvider: user.GitProvider,
		Updated:     now,
		Created:     now,
	}, nil
}
