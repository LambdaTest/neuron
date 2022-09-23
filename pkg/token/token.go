package token

import (
	"context"
	"net/http"
	"time"

	"github.com/LambdaTest/neuron/config"
	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/drone/go-scm/scm/transport/oauth2"
)

type gitTokenHandler struct {
	tokenRefresher gitTokenRefresher
	vaultStore     core.Vault
	githubApp      core.GithubApp
	logger         lumber.Logger
}

// gitTokenRefresher that returns go scm oauth refreshers
type gitTokenRefresher struct {
	bitbucketRefresher oauth2.Refresher
	// githubRefresher oauth2.Refresher
	gitlabRefresher oauth2.Refresher
}

func New(cfg *config.Config, vaultStore core.Vault, githubApp core.GithubApp, logger lumber.Logger) core.GitTokenHandler {
	return &gitTokenHandler{
		logger:     logger,
		vaultStore: vaultStore,
		githubApp:  githubApp,
		tokenRefresher: gitTokenRefresher{bitbucketRefresher: provideBitbucketRefresher(cfg, logger),
			gitlabRefresher: provideGitlabRefresher(cfg, logger)},
	}
}

func (g *gitTokenHandler) GetToken(
	ctx context.Context, getInstallation bool,
	gitProvider core.SCMDriver,
	k8namespace, orgName, mask, id string) (*core.Token, bool, error) {
	tokenPath := g.vaultStore.GetTokenPath(gitProvider, mask, id)

	var installationTokenPath string
	if getInstallation {
		installationTokenPath = g.vaultStore.GetInstallationTokenPath(gitProvider, k8namespace, orgName)
	}

	return g.GetTokenFromPath(ctx, gitProvider, tokenPath, installationTokenPath)
}

func (g *gitTokenHandler) GetTokenFromPath(ctx context.Context, gitProvider core.SCMDriver,
	tokenPath, installationTokenPath string) (*core.Token, bool, error) {
	var refreshed bool
	secret, err := g.vaultStore.ReadSecret(tokenPath)
	if err != nil {
		g.logger.Errorf("failed to list secret for path %s, error %v", tokenPath, err)
		return nil, refreshed, err
	}

	token, err := g.getTokenFromSecret(secret)
	if err != nil {
		g.logger.Errorf("error while parsing secret for token from path %s, %v", tokenPath, err)
		return nil, refreshed, err
	}

	token, refreshed, err = g.refreshToken(ctx, gitProvider, tokenPath, token)
	if err != nil {
		g.logger.Errorf("error while refreshing token for path %s, %v", tokenPath, err)
		return nil, refreshed, err
	}

	if gitProvider == core.DriverGithub && installationTokenPath != "" {
		token, err = g.getInstllationTokenFromPath(installationTokenPath, token)
		if err != nil {
			g.logger.Errorf("error while fetching installation token for path %s, %v", tokenPath, err)
			return token, refreshed, err
		}
	}

	return token, refreshed, nil
}

func (g *gitTokenHandler) getInstllationTokenFromPath(tokenPath string, token *core.Token) (*core.Token, error) {
	secret, err := g.vaultStore.ReadSecret(tokenPath)
	if err != nil {
		g.logger.Errorf("failed to list secret for path %s, error %v", tokenPath, err)
		return nil, err
	}

	installationToken, err := g.getInstallationTokenFromSecret(secret)
	if err != nil {
		g.logger.Errorf("error while parsing secret for installation token from path %s, %v", tokenPath, err)
		return nil, err
	}

	installationToken, err = g.refreshInstallationToken(installationToken, tokenPath)
	if err != nil {
		g.logger.Errorf("error while refreshing installation token for path %s, %v", tokenPath, err)
		return nil, err
	}

	token.InstallationID = installationToken.InstallationID
	token.InstallationExpiry = installationToken.InstallationExpiry
	token.InstallationToken = installationToken.InstallationToken

	return token, nil
}

// refreshToken returns a valid token. If current token has been expired then refreshes it
// and updates it in the vault for current user and returns the refreshed token else returns the same token back
func (g *gitTokenHandler) refreshToken(
	ctx context.Context,
	gitProvider core.SCMDriver,
	tokenPath string,
	token *core.Token) (*core.Token, bool, error) {
	var tokenRefreshed bool
	refresher, err := g.getRefresher(gitProvider)
	if err != nil {
		if err == errs.ErrUnsupportedGitProvider {
			return token, tokenRefreshed, nil
		}
		return nil, tokenRefreshed, err
	}

	ctx = token.SetRequestContext(ctx)
	// Token returns the new token if it hasn't been expired or about to with in the buffer time(15 mins)
	// else returns the same token
	scmToken, err := refresher.Token(ctx)
	if err != nil {
		g.logger.Errorf("error while refreshing token for git provider %s, %v", gitProvider, err)
		return nil, tokenRefreshed, err
	}

	// Updates the token if it has been renewed
	if scmToken.Token != token.AccessToken {
		token = &core.Token{
			AccessToken:  scmToken.Token,
			RefreshToken: scmToken.Refresh,
			Expiry:       scmToken.Expires,
		}
		if err := g.UpdateTokenInVault(tokenPath, token); err != nil {
			// In this case new token hasn't been updated in vault
			// but current task will still work with new token
			g.logger.Warnf("failed to update secret in vault storefor path %s: %v", tokenPath, err)
			return token, tokenRefreshed, nil
		}
		tokenRefreshed = true
	}

	return token, tokenRefreshed, nil
}

// refreshInstallationToken returns a valid instllation token. If current token has been expired then refreshes it
// and updates it in the vault for current org and returns the refreshed token else returns the same token back
func (g *gitTokenHandler) refreshInstallationToken(
	token *core.Token, tokenPath string) (*core.Token, error) {
	if expired := g.installationTokenExpired(token); !expired {
		return token, nil
	}

	_, installationToken, err := g.githubApp.GetInstallationToken(token.InstallationID)
	if err != nil {
		g.logger.Errorf("error while getting installation token, %v", err)
		return token, err
	}

	if err := g.updateInstallationTokenInVault(tokenPath, installationToken); err != nil {
		// In this case new token hasn't been updated in vault
		// but current task will still work with new token
		g.logger.Warnf("failed to update secret in vault store for path %s: %v", tokenPath, err)
		return installationToken, nil
	}

	return installationToken, nil
}

// UpdateTokenInVault updates token in vault only for current org
func (g *gitTokenHandler) UpdateTokenInVault(tokenPath string, token *core.Token) error {
	options := map[string]interface{}{
		"data": map[string]interface{}{
			"access_token":  token.AccessToken,
			"refresh_token": token.RefreshToken,
			"expiry":        token.Expiry,
		},
	}
	return g.vaultStore.CreateSecret(tokenPath, options)
}

// UpdateInstallationToken updates installation token in vault only for current org
func (g *gitTokenHandler) UpdateInstallationToken(gitProvider core.SCMDriver,
	k8namespace, orgName string, token *core.Token) error {
	tokenPath := g.vaultStore.GetInstallationTokenPath(gitProvider, k8namespace,
		orgName)

	g.logger.Debugf("tokenPath %s", tokenPath)

	return g.updateInstallationTokenInVault(tokenPath, token)
}

// updateInstallationTokenInVault updates installation token in vault only for current org
func (g *gitTokenHandler) updateInstallationTokenInVault(tokenPath string, token *core.Token) error {
	options := map[string]interface{}{
		"data": map[string]interface{}{
			"installation_token":  token.InstallationToken,
			"installation_expiry": token.InstallationExpiry,
			"installation_id":     token.InstallationID,
		},
	}
	return g.vaultStore.CreateSecret(tokenPath, options)
}

// getTokenFromSecret parses secret to get token
func (g *gitTokenHandler) getTokenFromSecret(secret map[string]interface{}) (*core.Token, error) {
	accessToken, ok := secret["access_token"].(string)
	if !ok {
		return nil, errs.ErrTypeAssertionFailed
	}

	refreshToken, ok := secret["refresh_token"].(string)
	if !ok {
		return nil, errs.ErrTypeAssertionFailed
	}

	expiryStr, ok := secret["expiry"].(string)
	if !ok {
		return nil, errs.ErrTypeAssertionFailed
	}

	expiry, err := time.Parse(time.RFC3339, expiryStr)
	if err != nil {
		return nil, err
	}

	return &core.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Expiry:       expiry,
	}, nil
}

// getInstallationTokenFromSecret parses secret to get insatllation token
func (g *gitTokenHandler) getInstallationTokenFromSecret(secret map[string]interface{}) (*core.Token, error) {
	installationToken, ok := secret["installation_token"].(string)
	if !ok {
		return nil, errs.ErrTypeAssertionFailed
	}

	installationID, ok := secret["installation_id"].(string)
	if !ok {
		return nil, errs.ErrTypeAssertionFailed
	}

	installationExpiryStr, ok := secret["installation_expiry"].(string)
	if !ok {
		return nil, errs.ErrTypeAssertionFailed
	}

	installationExpiry, err := time.Parse(time.RFC3339, installationExpiryStr)
	if err != nil {
		return nil, err
	}

	return &core.Token{
		InstallationToken:  installationToken,
		InstallationExpiry: installationExpiry,
		InstallationID:     installationID,
	}, nil
}

// provideBitbucketRefresher returns a Bitbucket Oauth Refresher
// based on the environment configuration
func provideBitbucketRefresher(cfg *config.Config, logger lumber.Logger) oauth2.Refresher {
	if cfg.Bitbucket.ClientID == "" {
		logger.Fatalf("missing Bitbucket ClientID")
	}
	if cfg.Bitbucket.ClientSecret == "" {
		logger.Fatalf("missing Bitbucket ClientSecret")
	}
	if cfg.Bitbucket.RefreshTokenEndpoint == "" {
		logger.Fatalf("missing Bitbucket RefreshTokenEndpoint")
	}

	return oauth2.Refresher{
		ClientID:     cfg.Bitbucket.ClientID,
		ClientSecret: cfg.Bitbucket.ClientSecret,
		Endpoint:     cfg.Bitbucket.RefreshTokenEndpoint,
		Source:       oauth2.ContextTokenSource(),
		Client:       &http.Client{},
	}
}

// provideGitlabRefresher returns a Gitlab Oauth Refresher
// based on the environment configuration
func provideGitlabRefresher(cfg *config.Config, logger lumber.Logger) oauth2.Refresher {
	if cfg.GitLab.ClientID == "" {
		logger.Fatalf("missing Gitlab ClientID")
	}
	if cfg.GitLab.ClientSecret == "" {
		logger.Fatalf("missing Gitlab ClientSecret")
	}
	if cfg.GitLab.RefreshTokenEndpoint == "" {
		logger.Fatalf("missing Gitlab RefreshTokenEndpoint")
	}

	return oauth2.Refresher{
		ClientID:     cfg.GitLab.ClientID,
		ClientSecret: cfg.GitLab.ClientSecret,
		Endpoint:     cfg.GitLab.RefreshTokenEndpoint,
		Source:       oauth2.ContextTokenSource(),
		Client:       &http.Client{},
	}
}

// installationTokenExpired reports whether the installation token is expired.
func (g *gitTokenHandler) installationTokenExpired(token *core.Token) bool {
	if token.InstallationExpiry.IsZero() && token.InstallationToken != "" {
		return false
	}

	return token.InstallationExpiry.Add(-constants.ExpiryDelta).
		Before(time.Now())
}

// GetRefresher returns the oauth refresher
func (g *gitTokenHandler) getRefresher(clientID core.SCMDriver) (oauth2.Refresher, error) {
	switch clientID {
	case core.DriverBitbucket:
		return g.tokenRefresher.bitbucketRefresher, nil
	case core.DriverGitlab:
		return g.tokenRefresher.gitlabRefresher, nil
	case core.DriverGithub:
		return oauth2.Refresher{}, errs.ErrUnsupportedGitProvider
	default:
		return oauth2.Refresher{}, errs.ErrInvalidDriver
	}
}
