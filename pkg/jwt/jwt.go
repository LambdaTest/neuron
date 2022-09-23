package jwt

import (
	"crypto/rsa"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/LambdaTest/neuron/config"
	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	jsoniter "github.com/json-iterator/go"
)

const (
	defaultSigningAlgo     = "RS256"
	defaultTokenHeaderName = "Bearer"
	defaultCookieName      = "_session_"
	defaultTimeout         = time.Hour * 24 * 30
	repoIDKey              = "repoID"
)

// JWTAuthorizer provides a Json-Web-Token authentication implementation. On failure, a 401 HTTP response
// is returned.
type JWTAuthorizer struct {
	// signing algorithm - possible values are HS256, HS384, HS512, RS256, RS384 or RS512
	signingAlgorithm string
	// Duration that a jwt token is valid. Optional, defaults to one hour.
	timeout time.Duration
	// TokenHeadName is a string in the header. Default value is "Bearer"
	tokenHeadName string
	// Private key
	privKey *rsa.PrivateKey
	// Public key
	pubKey *rsa.PublicKey
	// the logger object
	logger lumber.Logger
	// Duration that a cookie is valid. Optional, by default equals to Timeout value.
	cookieMaxAge time.Duration
	// Allow insecure cookies for development over http
	secureCookie bool
	// Allow cookie domain change for development
	cookieDomain string
	// cookieName allow cookie name change for development
	cookieName string
	// cookieSameSite allow use http.SameSite cookie param
	cookieSameSite http.SameSite
}

// New returns a new session authorizer
func New(cfg *config.Config, logger lumber.Logger) (core.Session, error) {
	privKeyBytes, err := base64.StdEncoding.DecodeString(cfg.JWT.PrivateKey)
	if err != nil {
		logger.Errorf("error while b64 decoding private key %v", err)
		return nil, err
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privKeyBytes)
	if err != nil {
		logger.Errorf("error while parsing RSA private key %v", err)
		return nil, errors.ErrInvalidPrivKey
	}
	pubKeyBytes, err := base64.StdEncoding.DecodeString(cfg.JWT.PublicKey)
	if err != nil {
		logger.Errorf("error while b64 decoding public key %v", err)
		return nil, err
	}

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(pubKeyBytes)
	if err != nil {
		logger.Errorf("error while parsing RSA public key %v", err)
		return nil, errors.ErrInvalidPubKey
	}
	cookieName, ok := constants.CookieName[cfg.Env]
	if !ok {
		cookieName = defaultCookieName
	}

	timeout := cfg.JWT.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	domain := constants.CookieDomain[cfg.Env]
	secureCookie := true
	if cfg.Env == constants.Dev {
		secureCookie = false
	}

	return &JWTAuthorizer{
		signingAlgorithm: defaultSigningAlgo,
		tokenHeadName:    defaultTokenHeaderName,
		timeout:          timeout,
		privKey:          privateKey,
		pubKey:           publicKey,
		cookieName:       cookieName,
		cookieDomain:     domain,
		cookieMaxAge:     timeout,
		cookieSameSite:   http.SameSiteLaxMode,
		secureCookie:     secureCookie,
		logger:           logger,
	}, nil
}

// CreateInternalToken creates internal JWT token
func (jw *JWTAuthorizer) CreateTokenInternal(data *core.BuildData) (string, error) {
	claims := NewJWTClaims()
	claims.SetIssuedAt(time.Now().Unix())
	claims.SetExpiry(time.Now().Add(jw.timeout).Unix())
	claims.SetJTI(utils.GenerateUUID())

	if err := setCustomClaimInternal(claims, data); err != nil {
		return "", err
	}
	token := jwt.NewWithClaims(jwt.GetSigningMethod(jw.signingAlgorithm), claims)

	tokenString, err := token.SignedString(jw.privKey)
	if err != nil {
		jw.logger.Errorf("failed to create JWT token, error %v", err)
	}

	return tokenString, nil
}

// CreateToken method that clients can use to create a jwt token and returns the http cookie.
func (jw *JWTAuthorizer) CreateToken(data *core.UserData) (*http.Cookie, error) {
	claims := NewJWTClaims()
	claims.SetIssuedAt(time.Now().Unix())
	claims.SetExpiry(time.Now().Add(jw.timeout).Unix())
	claims.SetJTI(utils.GenerateUUID())

	if err := claims.SetCustomClaim(data); err != nil {
		return nil, err
	}
	token := jwt.NewWithClaims(jwt.GetSigningMethod(jw.signingAlgorithm), claims)

	tokenString, err := token.SignedString(jw.privKey)
	if err != nil {
		jw.logger.Errorf("failed to create JWT token, error %v", err)
		return nil, errors.ErrFailedTokenCreation
	}

	expireCookie := time.Now().Add(jw.cookieMaxAge)
	maxage := int(expireCookie.Unix() - time.Now().Unix())

	return &http.Cookie{
		Name:     jw.cookieName,
		Value:    tokenString,
		Domain:   jw.cookieDomain,
		Secure:   jw.secureCookie,
		MaxAge:   maxage,
		Path:     "/",
		SameSite: jw.cookieSameSite,
	}, nil
}

// DeleteCookie deletes the cookie.
func (jw *JWTAuthorizer) DeleteCookie() *http.Cookie {
	return &http.Cookie{
		Name:     jw.cookieName,
		Value:    "",
		Domain:   jw.cookieDomain,
		Secure:   jw.secureCookie,
		MaxAge:   -1,
		Path:     "/",
		SameSite: jw.cookieSameSite,
	}
}

// Authorize vaildates and extracts the data from JWT Token claims from request header
func (jw *JWTAuthorizer) Authorize(c *gin.Context) (*core.UserData, error) {
	var token string
	repoID := c.GetString(repoIDKey)
	authHeader := c.Request.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if !(len(parts) == 2 && parts[0] == jw.tokenHeadName) {
			jw.logger.Errorf("Error while parsing auth token, got Authorization token: %s", authHeader)
			if repoID != "" {
				c.AbortWithStatusJSON(http.StatusNotFound, errors.ErrNotFound)
			} else {
				c.AbortWithStatusJSON(http.StatusForbidden, errors.ErrInvalidAuthHeader)
			}
			return nil, errors.ErrInvalidAuthHeader
		}
		token = parts[1]
	} else {
		cookie, _ := c.Cookie(jw.cookieName)
		token = cookie
	}

	if token == "" {
		if repoID != "" {
			c.AbortWithStatusJSON(http.StatusNotFound, errors.ErrNotFound)
		} else {
			c.AbortWithStatusJSON(http.StatusForbidden, errors.ErrMissingToken)
		}
		return nil, errors.ErrMissingToken
	}

	claims, err := jw.parseToken(token)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, err)
		return nil, err
	}

	userData, err := jw.extractData(claims)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, err)
		return nil, err
	}

	return userData, err
}

// AuthorizeInternal parses and validates the internal JWT Token
func (jw *JWTAuthorizer) AuthorizeInternal(c *gin.Context) (*core.BuildData, error) {
	var token string
	authHeader := c.Request.Header.Get("Authorization")
	if authHeader == "" {
		c.AbortWithStatusJSON(http.StatusForbidden, errors.ErrMissingToken)
		return nil, errors.ErrMissingToken
	}
	parts := strings.Split(authHeader, " ")
	if !(len(parts) == 2 && parts[0] == jw.tokenHeadName) {
		jw.logger.Errorf("Error while parsing auth token, got Authorization token: %s", authHeader)
		c.AbortWithStatusJSON(http.StatusForbidden, errors.ErrInvalidAuthHeader)
		return nil, errors.ErrInvalidAuthHeader
	}
	token = parts[1]

	if token == "" {
		c.AbortWithStatusJSON(http.StatusForbidden, errors.ErrMissingToken)
		return nil, errors.ErrMissingToken
	}

	claims, err := jw.parseTokenInternal(token)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, err)
		return nil, err
	}

	buildData, err := jw.extractDataInternal(claims)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, err)
		return nil, err
	}

	return buildData, err
}

func (jw *JWTAuthorizer) parseTokenInternal(token string) (*JwtClaims, error) {
	jwtToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if jwt.GetSigningMethod(jw.signingAlgorithm) != t.Method {
			return nil, errors.ErrInvalidSigningAlgorithm
		}
		return jw.pubKey, nil
	})
	if err != nil {
		jw.logger.Errorf("error while parsing jwt token, error: %v", err)
		return nil, errors.ErrInvalidToken
	}

	// extract claims from token
	mapClaims := jwtToken.Claims.(jwt.MapClaims)

	jc := NewJWTClaims()
	jc.MapClaims = mapClaims

	// check if claims are valid
	if err := validInternal(jc); err != nil {
		jw.logger.Errorf("error while parsing jwt token, error: %v", err)
		return nil, err
	}

	return jc, nil
}

func (jw *JWTAuthorizer) extractDataInternal(jc *JwtClaims) (*core.BuildData, error) {
	rawBytes, err := jc.MarshalJSON()
	if err != nil {
		jw.logger.Errorf("failed to marshall jwt claim payload, error:%v", err)
		return nil, errors.ErrMarshalJSON
	}
	buildData := new(core.BuildData)

	json := jsoniter.ConfigCompatibleWithStandardLibrary
	if err = json.Unmarshal(rawBytes, &buildData); err != nil {
		jw.logger.Errorf("failed to unmarshall jwt claim payload, error:%v", err)
		return nil, errors.ErrUnMarshalJSON
	}

	return buildData, nil
}

func (jw *JWTAuthorizer) parseToken(token string) (*JwtClaims, error) {
	jwtToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if jwt.GetSigningMethod(jw.signingAlgorithm) != t.Method {
			return nil, errors.ErrInvalidSigningAlgorithm
		}
		return jw.pubKey, nil
	})
	if err != nil {
		jw.logger.Errorf("error while parsing jwt token, error: %v", err)
		return nil, errors.ErrInvalidToken
	}

	// extract claims from token
	mapClaims := jwtToken.Claims.(jwt.MapClaims)

	jc := NewJWTClaims()
	jc.MapClaims = mapClaims

	// check if claims are valid
	if err := jc.Valid(); err != nil {
		jw.logger.Errorf("error while parsing jwt token, error: %v", err)
		return nil, err
	}

	return jc, nil
}

func (jw *JWTAuthorizer) extractData(jc *JwtClaims) (*core.UserData, error) {
	rawBytes, err := jc.MarshalJSON()
	if err != nil {
		jw.logger.Errorf("failed to marshall jwt claim payload, error:%v", err)
		return nil, errors.ErrMarshalJSON
	}
	sessionData := new(core.UserData)

	json := jsoniter.ConfigCompatibleWithStandardLibrary
	if err = json.Unmarshal(rawBytes, &sessionData); err != nil {
		jw.logger.Errorf("failed to unmarshall jwt claim payload, error:%v", err)
		return nil, errors.ErrUnMarshalJSON
	}

	return sessionData, nil
}

func validInternal(c *JwtClaims) error {
	now := time.Now().Unix()

	if !c.MapClaims.VerifyExpiresAt(now, true) {
		return errors.ErrExpiredToken
	}
	if !c.MapClaims.VerifyIssuedAt(now, true) {
		return errors.ErrExpiredToken
	}

	if _, ok := c.MapClaims["jti"].(string); !ok {
		return errors.ErrMissingJTI
	}

	if _, ok := c.MapClaims["build_id"].(string); !ok {
		return errors.ErrMissingBuildIDClaim
	}

	if _, ok := c.MapClaims["org_id"].(string); !ok {
		return errors.ErrMissingOrgIDClaim
	}

	if _, ok := c.MapClaims["repo_id"].(string); !ok {
		return errors.ErrMissingRepoIDClaim
	}

	return nil
}

func setCustomClaimInternal(c *JwtClaims, data *core.BuildData) error {
	if data.BuildID == "" || data.OrgID == "" || data.RepoID == "" {
		return errors.ErrMissingBuildData
	}
	c.MapClaims["build_id"] = data.BuildID
	c.MapClaims["org_id"] = data.OrgID
	c.MapClaims["repo_id"] = data.RepoID
	return nil
}
