package core

import (
	"context"
	"time"

	"github.com/drone/go-scm/scm"
)

// Token represents the git oauth token for private repositories
type Token struct {
	AccessToken        string    `json:"access_token"`
	RefreshToken       string    `json:"refresh_token,omitempty"`
	Expiry             time.Time `json:"expiry"`
	InstallationToken  string    `json:"installation_token,omitempty"`
	InstallationExpiry time.Time `json:"installation_expiry,omitempty"`
	InstallationID     string    `json:"installation_id,omitempty"`
}

// GitTokenHandler handles git token related things
type GitTokenHandler interface {
	// GetToken returns a valid token for given args
	GetToken(ctx context.Context, getInstallation bool,
		gitProvider SCMDriver, k8namespace, orgName, mask, id string) (token *Token, tokenRefreshed bool, err error)
	// GetTokenFromPath returns a valid token for given token path
	GetTokenFromPath(ctx context.Context, gitProvider SCMDriver,
		tokenPath, installationTokenPath string) (token *Token, tokenRefreshed bool, err error)
	// UpdateInstallationToken updates installation token in vault
	UpdateInstallationToken(gitProvider SCMDriver, k8namespace, orgName string, token *Token) error
	// UpdateTokenInVault updates oauth token in vault
	UpdateTokenInVault(tokenPath string, token *Token) error
}

// SetRequestContext sets the token values in the request context
func (t *Token) SetRequestContext(ctx context.Context) context.Context {
	token := &scm.Token{
		Token:   t.AccessToken,
		Refresh: t.RefreshToken,
		Expires: t.Expiry,
	}

	return context.WithValue(ctx, scm.TokenKey{}, token)
}

// SetInstallationRequestContext sets the installation token values in the request context
func (t *Token) SetInstallationRequestContext(ctx context.Context) context.Context {
	if t.InstallationToken == "" {
		return t.SetRequestContext(ctx)
	}
	token := &scm.Token{
		Token:   t.InstallationToken,
		Refresh: "",
		Expires: t.InstallationExpiry,
	}

	return context.WithValue(ctx, scm.TokenKey{}, token)
}
