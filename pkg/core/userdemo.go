package core

import (
	"context"
)

// UserInfoDemoDetails is a request body of user information
type UserInfoDemoDetails struct {
	ID        string `db:"id"`
	Name      string `db:"name"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name"`
	EmailID   string `db:"email_id" json:"email_id" binding:"required,email"`
	Company   string `db:"company_name" json:"company"`
	Created   string `db:"created_at" json:"-"`
	Updated   string `db:"updated_at" json:"-"`
}

// UserDemoStore defines datastore operation for working with userdemo related data.
type UserDemoStore interface {
	// Create creates the user info in the db
	Create(ctx context.Context, info *UserInfoDemoDetails) error
}
