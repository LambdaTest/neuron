package gituser

import (
	"context"

	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/drone/go-login/login"
)

type service struct {
	scmProvider core.SCMProvider
	logger      lumber.Logger
}

// New returns a new User service that provides access to
// user data from the source code management system.
func New(scmProvider core.SCMProvider, logger lumber.Logger) core.GitUserService {
	return &service{scmProvider: scmProvider, logger: logger}
}

// Find returns the authenticated user.
func (s *service) Find(ctx context.Context, driver core.SCMDriver, loginToken *login.Token) (*core.GitUser, error) {
	gitSCM, err := s.scmProvider.GetClient(driver)
	if err != nil {
		s.logger.Errorf("failed to find git client %v", err)
		return nil, err
	}

	oauth := core.Token{
		AccessToken:  loginToken.Access,
		RefreshToken: loginToken.Refresh,
		Expiry:       loginToken.Expires,
	}

	// set token in context for the git scm client
	ctx = oauth.SetRequestContext(ctx)

	src, _, err := gitSCM.Client.Users.Find(ctx)
	if err != nil {
		s.logger.Errorf("error while finding user %v", err)
		return nil, err
	}

	if src.Email == "" {
		email, _, err := gitSCM.Client.Users.FindEmail(ctx)
		if err != nil {
			s.logger.Errorf("error while finding user email %v", err)
			return nil, err
		}
		src.Email = email
	}

	// changing username to email in case of bitbucket
	if driver == core.DriverBitbucket {
		src.Login = src.Email
	}

	return &core.GitUser{
		ID:          utils.GenerateUUID(),
		Avatar:      src.Avatar,
		Username:    src.Login,
		Email:       src.Email,
		Mask:        utils.RandString(constants.BitSize16),
		GitProvider: driver,
		Oauth:       oauth,
	}, nil
}
