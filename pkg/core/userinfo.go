package core

import (
	"context"
	"time"
)

// UserInfo represents the information entity of user
type UserInfo struct {
	ID              string    `db:"id"`
	UserID          string    `db:"user_id"`
	UserDescription string    `db:"user_description"`
	Experience      string    `db:"experience"`
	TeamSize        string    `db:"team_size"`
	OrgID           string    `db:"org_id"`
	Created         time.Time `db:"created_at"`
	Updated         time.Time `db:"updated_at"`
	IsActive        bool      `db:"is_active"`
}

// OpenSourceUserInfo is a request body of a open source user
type OpenSourceUserInfo struct {
	FirstName   string `json:"first_name" binding:"required"`
	LastName    string `json:"last_name"`
	EmailID     string `json:"email_id" binding:"required"`
	Link        string `json:"repository_link" binding:"required"`
	Description string `json:"description"`
}

// UserInfoStore defines datastore operation for working with userInfo related data.
type UserInfoStore interface {
	// Create adds the information of user in database
	Create(ctx context.Context, userInfo *UserInfo) error
	// Find finds the info of a user from a database
	Find(ctx context.Context, userID, orgID string) (*UserInfo, error)
	// UpdateActiveUser adds the information of user in database
	UpdateActiveUser(ctx context.Context, userInfo *UserInfo) error
}
