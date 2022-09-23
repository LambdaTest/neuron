package githubapp

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/LambdaTest/neuron/config"
	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/errors"
	jwtClaims "github.com/LambdaTest/neuron/pkg/jwt"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/golang-jwt/jwt/v4"
)

type CtxKey string

type installationOutput struct {
	Account struct {
		Login string `json:"login"`
	} `json:"account"`
}

type accessTokenOutput struct {
	Token  string    `json:"token"`
	Expiry time.Time `json:"expires_at"`
}

type githubApp struct {
	logger     lumber.Logger
	requests   core.Requests
	privateKey *rsa.PrivateKey
	appID      string
}

func New(cfg *config.Config, requests core.Requests, logger lumber.Logger) (core.GithubApp, error) {
	if cfg.GitHub.AppID == "" {
		logger.Errorf("Missing Github App ID")
		return nil, errors.ErrConfigNotFound
	}

	if cfg.GitHub.PrivateKey == "" {
		logger.Errorf("Missing Github App PrivateKey")
		return nil, errors.ErrConfigNotFound
	}

	privKeyBytes, err := base64.StdEncoding.DecodeString(cfg.GitHub.PrivateKey)
	if err != nil {
		logger.Errorf("error while b64 decoding private key for github app %v", err)
		return nil, err
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privKeyBytes)
	if err != nil {
		logger.Errorf("error while parsing RSA private key for github app %v", err)
		return nil, errors.ErrInvalidPrivKey
	}

	return &githubApp{
		logger:     logger,
		requests:   requests,
		privateKey: privateKey,
		appID:      cfg.GitHub.AppID,
	}, nil
}

func (g *githubApp) createJWT() (string, error) {
	claims := jwtClaims.NewJWTClaims()
	claims.SetIssuedAt(time.Now().Unix())
	claims.SetExpiry(time.Now().Add(constants.GithubAppJWTExpiryDuration).Unix())
	claims.SetIssuer(g.appID)

	token := jwt.NewWithClaims(jwt.GetSigningMethod("RS256"), claims)

	return token.SignedString(g.privateKey)
}

func (g *githubApp) GetInstallationToken(installationID string) (string, *core.Token, error) {
	jwtToken, err := g.createJWT()
	if err != nil {
		g.logger.Errorf("failed to create JWT token for github app, error %v", err)
		return "", nil, errors.ErrFailedTokenCreation
	}

	tokenEndPoint := fmt.Sprintf("%s/installations/%s/access_tokens", constants.GithubAppBaseURL, installationID)

	resBytes, err := g.requests.MakeAPIRequest(context.TODO(), "POST", tokenEndPoint, nil, jwtToken)
	if err != nil {
		g.logger.Errorf("Error while making get installation token api request : %s", err)
		return "", nil, err
	}

	out := new(accessTokenOutput)
	err = json.Unmarshal(resBytes, &out)
	if err != nil {
		g.logger.Errorf("error while unmarshaling json to accessTokenOutput  : err", err)
		return "", nil, err
	}

	return jwtToken, &core.Token{InstallationToken: out.Token, InstallationID: installationID, InstallationExpiry: out.Expiry}, nil
}

func (g *githubApp) GetInstallation(installationID string) (string, *core.Token, error) {
	jwtToken, token, err := g.GetInstallationToken(installationID)
	if err != nil {
		g.logger.Errorf("failed to get installation access token for github app, error %v", err)
		return "", nil, errors.ErrFailedTokenCreation
	}

	installationEndPoint := fmt.Sprintf("%s/installations/%s", constants.GithubAppBaseURL, installationID)

	resBytes, err := g.requests.MakeAPIRequest(context.TODO(), "GET", installationEndPoint, nil, jwtToken)
	if err != nil {
		g.logger.Errorf("Error while making get installation api request : %s", err)
		return "", nil, err
	}

	out := new(installationOutput)
	err = json.Unmarshal(resBytes, &out)
	if err != nil {
		g.logger.Errorf("error while unmarshaling json to installationOutput  : err", err)
		return "", nil, err
	}
	return out.Account.Login, token, nil
}
