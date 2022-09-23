package emailnotifications

import (
	"bytes"
	"context"
	"fmt"
	"html/template"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/errs"
)

const (
	repoImportSubject  = "New repo linked with TAS successfully."
	repoImportTemplate = "static/mailtemplates/newrepo.html"
)

func (e *emailNotificationManager) SendRepoNotification(ctx context.Context, repo *core.Repository) error {
	user, err := e.gitUserStore.FindByID(ctx, repo.Admin)
	if err != nil {
		e.logger.Errorf("can't find admin in userStore for the repo: %v", repo.Name)
		return errs.New("admin details not found")
	}

	orgID := repo.OrgID
	organization, err := e.orgStore.FindByID(ctx, orgID)
	if err != nil {
		e.logger.Errorf("could not find organization for repo: %s, error: %v", repo.Name, err)
		return err
	}

	addresses, err := e.validateAddresses(map[string]string{user.Email: user.Username})
	if err != nil {
		e.logger.Errorf("error in validating email addresses: %v, error: %v", user.Email, err)
		return err
	}

	templateFile := repoImportTemplate
	parsed, err := template.ParseFiles(templateFile)
	if err != nil {
		e.logger.Errorf("failed to parse %v error: %v", templateFile, err)
		return err
	}
	adminRepoDashboard := fmt.Sprintf("%s/%s/%s/%s/jobs/", e.cfg.FrontendURL, string(user.GitProvider), organization.Name, repo.Name)

	var body bytes.Buffer
	if errE := parsed.Execute(&body, templateMetaData{
		AdminUserName: user.Username,
		Project:       repo.Name,
		DashboardURL:  adminRepoDashboard,
		ImageURL:      e.rocketImageURL,
	}); errE != nil {
		e.logger.Errorf("failed to update dynamic data in html body, %v", errE)
		return errE
	}

	emailData := []Message{
		{
			From:    e.cfg.SenderEmail,
			To:      addresses,
			Subject: repoImportSubject,
			Body:    body.String(),
		},
	}
	if err := e.sendBatchEmails(ctx, emailData, e.endpoint); err != nil {
		e.logger.Errorf("error in sending repo import email to %s: %v", user.Email, err)
		return err
	}

	e.logger.Debugf("new repo: %s import notification email send to %s", repo.Name, user.Email)
	return nil
}
