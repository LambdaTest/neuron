package emailnotifications

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
)

const (
	firstPassedBuildSubject  = "Great! First job executed successfully on TAS."
	firstPassedBuildTemplate = "static/mailtemplates/passedbuild.html"
	firstFailedBuildTemplate = "static/mailtemplates/failedbuild.html"
)

func (e *emailNotificationManager) SendBuildStatus(ctx context.Context, orgName,
	buildID, gitProvider, repoName, totalTests string, passedBuild bool) error {
	admin, err := e.gitUserStore.FindAdminByBuildID(ctx, buildID)
	if err != nil {
		e.logger.Errorf("admin data could not be fetched, error: %+v\n", err)
		return err
	}
	if admin.Email == "" {
		e.logger.Debugf("user email not found, aborting to send build notification mail")
		return nil
	}

	adminAddress := map[string]string{admin.Email: admin.Username}
	address, err := e.validateAddresses(adminAddress)
	if err != nil {
		e.logger.Errorf("error in validating email addresses: %v, error: %v", admin.Email, err)
		return err
	}

	subject := ""
	templateFile := ""
	if passedBuild {
		subject = firstPassedBuildSubject
		templateFile = firstPassedBuildTemplate
	} else {
		subject = fmt.Sprintf("TAS : First test execution job on %v", repoName)
		templateFile = firstFailedBuildTemplate
	}
	parsed, err := template.ParseFiles(templateFile)
	if err != nil {
		e.logger.Errorf("failed to parse %v error: %v", templateFile, err)
		return err
	}

	dashboardURL := fmt.Sprintf("%s/%s/%s/%s/jobs/", e.cfg.FrontendURL, gitProvider, orgName, repoName)

	var body bytes.Buffer
	var errE error
	if passedBuild {
		errE = parsed.Execute(&body, templateMetaData{
			AdminUserName: admin.Username,
			Project:       repoName,
			TotalTests:    totalTests,
			DashboardURL:  dashboardURL,
			ImageURL:      e.rocketImageURL,
		})
	} else {
		errE = parsed.Execute(&body, templateMetaData{
			AdminUserName: admin.Username,
			Project:       repoName,
			DashboardURL:  dashboardURL,
			ImageURL:      e.rocketImageURL,
		})
	}
	if errE != nil {
		e.logger.Errorf("failed to update dynamic data in html body, %v", errE)
		return errE
	}

	emailData := []Message{
		{
			From:    e.cfg.SenderEmail,
			To:      address,
			Subject: subject,
			Body:    body.String(),
		},
	}
	if err := e.sendBatchEmails(ctx, emailData, e.endpoint); err != nil {
		e.logger.Errorf("error in sending build status email to %s: %v", admin.Email, err)
	}

	e.logger.Debugf("build status sent to %s", admin.Email)
	return nil
}
