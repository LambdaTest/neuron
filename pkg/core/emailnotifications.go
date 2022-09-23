package core

import (
	"context"
)

// EmailNotificationManager has contracts for sendmail
type EmailNotificationManager interface {
	// SendMailOnNewUserSignUp make api request to send email to new joinee, team notification, admin notification
	SendMailOnNewUserSignUp(ctx context.Context, user *GitUser, frontendURL string, orgs []*Organization) error
	// SendRepoNotification make api post request to send email to repo_admin on successful import of new repo
	SendRepoNotification(ctx context.Context, repo *Repository) error
	// UserDemoNotification make api post request to send email to TAS team when user take demo
	UserDemoNotification(ctx context.Context, user *UserInfoDemoDetails) error
	// SendBuildStatus sends email notification to repo admin on first successful (passed or failed) build
	SendBuildStatus(ctx context.Context, orgName, buildID,
		gitProvider, repoName, totalTests string, passedJob bool) error
	// OpenSourceUserNotification sends mail to TAS team in case open source user add details
	OpenSourceUserNotification(ctx context.Context, user *OpenSourceUserInfo) error
}
