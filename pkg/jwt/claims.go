package jwt

import (
	"time"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/errors"
	"github.com/golang-jwt/jwt/v4"
	jsoniter "github.com/json-iterator/go"
)

// JwtClaims - represents the jwt claims
type JwtClaims struct {
	jwt.MapClaims
}

// NewJWTClaims - Initializes a new jwt claims
func NewJWTClaims() *JwtClaims {
	return &JwtClaims{MapClaims: jwt.MapClaims{}}
}

// SetExpiry sets expiry in unix epoch secs
func (c *JwtClaims) SetExpiry(timeUnix int64) {
	c.MapClaims["exp"] = timeUnix
}

// SetIssuedAt sets expiry in unix epoch secs
func (c *JwtClaims) SetIssuedAt(timeUnix int64) {
	c.MapClaims["iat"] = timeUnix
}

// SetJTI sets sets the unique JWT ID
func (c *JwtClaims) SetJTI(jti string) {
	c.MapClaims["jti"] = jti
}

// SetIssuer sets the issuer for these claims
func (c *JwtClaims) SetIssuer(issuer string) {
	c.MapClaims["iss"] = issuer
}

// SetCustomClaim sets the custom claim in payload
func (c *JwtClaims) SetCustomClaim(data *core.UserData) error {
	if data.UserID == "" || data.UserName == "" || data.GitProvider == "" {
		return errors.ErrMissingUserData
	}
	c.MapClaims["user_id"] = data.UserID
	c.MapClaims["username"] = data.UserName
	c.MapClaims["git_provider"] = data.GitProvider
	return nil
}

// Valid Checks if the JWT Token is valid
func (c *JwtClaims) Valid() error {
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

	if _, ok := c.MapClaims["user_id"].(string); !ok {
		return errors.ErrMissingUserID
	}

	if _, ok := c.MapClaims["git_provider"].(string); !ok {
		return errors.ErrMissingGitProvider
	}

	if _, ok := c.MapClaims["username"].(string); !ok {
		return errors.ErrMissingUserName
	}

	return nil
}

// MarshalJSON marshals the MapClaims struct
func (c *JwtClaims) MarshalJSON() ([]byte, error) {
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	return json.Marshal(c.MapClaims)
}
