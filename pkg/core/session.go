package core

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Session provides session management for
// authenticated users.
type Session interface {
	// CreateToken creates the JWT token
	CreateToken(data *UserData) (*http.Cookie, error)
	// Authorize parses and validates the JWT Token
	Authorize(c *gin.Context) (*UserData, error)
	// CreateInternalToken creates internal JWT token
	CreateTokenInternal(data *BuildData) (string, error)
	// AuthorizeInternal parses and validates the internal JWT Token
	AuthorizeInternal(c *gin.Context) (*BuildData, error)
	// Invalidate deletes the cookie from server side.
	DeleteCookie() *http.Cookie
}

// UserData represents the data stored  in the cookies
type UserData struct {
	Expiry      int64     `json:"exp"`
	JwtID       string    `json:"jti"`
	UserID      string    `json:"user_id"`
	GitProvider SCMDriver `json:"git_provider"`
	UserName    string    `json:"username"`
}

// BuildData represents the data which is stored in JWT for internal auth
type BuildData struct {
	Expiry  int64  `json:"exp"`
	JwtID   string `json:"jti"`
	BuildID string `json:"build_id"`
	OrgID   string `json:"org_id"`
	RepoID  string `json:"repo_id"`
}
