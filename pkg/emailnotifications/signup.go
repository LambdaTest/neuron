package emailnotifications

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/mail"

	"github.com/LambdaTest/neuron/config"
	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/errs"
)

const (
	welcomeTemplate              = "static/mailtemplates/welcome.html"
	teamMailSubject              = "New User Signup on TAS"
	teamMailTemplate             = "static/mailtemplates/teaminternal.html"
	demoSubject                  = "New User visit for TAS Demo on login page"
	demoTemplate                 = "static/mailtemplates/demo_user_notification.html"
	newUserInExistingOrgSubject  = "New team member joined you on TAS"
	newUserInExistingOrgTemplate = "static/mailtemplates/newuserinorg.html"
	openSourceUserTemplate       = "static/mailtemplates/open_source_user.html"
	openSourceUserSubject        = "New Open Source User visit TAS for contribution"
)

type templateMetaData struct {
	UserName        string
	AdminUserName   string
	LoginMode       string
	TimeStamp       string
	Project         string
	TotalTests      string
	Env             string
	DashboardURL    string
	IndexPageURL    string
	ImageURL        string
	OrgName         string
	OrgDashboardURL string
}

type Message struct {
	From    string   `json:"from" validate:"required,email"`
	To      []string `json:"to" validate:"required,email"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
}

type multipleEmails struct {
	Messages []Message `json:"messages"`
}

type emailNotificationManager struct {
	cfg            *config.Config
	logger         lumber.Logger
	requests       core.Requests
	endpoint       string
	rocketImageURL string
	gitUserStore   core.GitUserStore
	orgStore       core.OrganizationStore
}

type orgAdminDetail struct {
	orgName    string
	adminName  string
	adminEmail string
}

// New returns a new EmailNotificationManager
func New(cfg *config.Config,
	gitUserStore core.GitUserStore,
	orgStore core.OrganizationStore,
	requests core.Requests,
	logger lumber.Logger) core.EmailNotificationManager {
	return &emailNotificationManager{
		cfg:            cfg,
		logger:         logger,
		orgStore:       orgStore,
		requests:       requests,
		endpoint:       fmt.Sprintf("%s/batchEmail", cfg.RavenRemoteHost),
		gitUserStore:   gitUserStore,
		rocketImageURL: fmt.Sprintf("%s/assets/images/emailer/rocket.png", cfg.FrontendURL),
	}
}

func (e *emailNotificationManager) SendMailOnNewUserSignUp(ctx context.Context,
	user *core.GitUser,
	frontendURL string,
	userOrgs []*core.Organization) error {
	// check the correctness of senderEmail
	if _, err := mail.ParseAddress(e.cfg.SenderEmail); err != nil {
		e.logger.Errorf("invalid sender email: %v, error: %v", e.cfg.SenderEmail, err)
		return err
	}

	signUpTime := e.getCurrentTimeInIST()
	if err := e.teamNotification(ctx, user.Username, user.GitProvider.String(), signUpTime); err != nil {
		// not returning error here because we need to send other emails on new user signup
		e.logger.Errorf("error in sending email to team, %v", err)
	}

	if user.Email != "" && user.Username != "" {
		if err := e.welcomeMail(ctx, user); err != nil {
			// not returning error here because we need to send other emails on new user signup
			e.logger.Errorf("error in sending welcome email to %s: %v", user.Username, err)
		}
	}

	admins := []orgAdminDetail{}
	for _, org := range userOrgs {
		admin, _ := e.gitUserStore.FindAdminByOrgNameAndGitProvider(ctx, org.Name, org.GitProvider.String())
		if admin.ID != "" && admin.ID != user.ID && admin.Email != "" {
			validAdminData := orgAdminDetail{
				orgName:    org.Name,
				adminName:  admin.Username,
				adminEmail: admin.Email,
			}
			admins = append(admins, validAdminData)
		}
	}

	if err := e.orgAdminsMail(ctx, user.Username, user.GitProvider, admins); err != nil {
		e.logger.Errorf("error in sending email to org admins: %v", err)
		return err
	}
	return nil
}

func (e *emailNotificationManager) UserDemoNotification(ctx context.Context, user *core.UserInfoDemoDetails) error {
	if _, err := mail.ParseAddress(e.cfg.SenderEmail); err != nil {
		e.logger.Errorf("invalid sender email: %v", err)
		return err
	}

	addresses, err := e.validateAddresses(e.cfg.TasTeam)
	if err != nil {
		e.logger.Errorf("error in validating email addresses: %v, error: %v", e.cfg.TasTeam, err)
		return err
	}

	templateFile := demoTemplate
	parsed, err := template.ParseFiles(templateFile)
	if err != nil {
		e.logger.Errorf("failed to parse %v error: %v", templateFile, err)
		return err
	}

	var body bytes.Buffer
	if errE := parsed.Execute(&body, struct {
		Name    string
		EmailID string
		Company string
	}{
		Name:    fmt.Sprintf("%s %s", user.FirstName, user.LastName),
		EmailID: user.EmailID,
		Company: user.Company,
	}); errE != nil {
		e.logger.Errorf("failed to update dynamic data in html body, %v", errE)
		return errE
	}

	emailData := []Message{
		{
			From:    e.cfg.SenderEmail,
			To:      addresses,
			Subject: demoSubject,
			Body:    body.String(),
		},
	}
	if err := e.sendBatchEmails(ctx, emailData, e.endpoint); err != nil {
		e.logger.Errorf("error in sending demo notification mail to tasTeam: %v", err)
	}

	e.logger.Debugf("demo signup notification mail send to team")
	return nil
}

func (e *emailNotificationManager) teamNotification(ctx context.Context, userName, loginMode, timeStamp string) error {
	tasTeam := e.cfg.TasTeam
	addresses, err := e.validateAddresses(tasTeam)
	if err != nil {
		e.logger.Errorf("error in validating email addresses: %v, error: %v", tasTeam, err)
		return err
	}

	templateFile := teamMailTemplate
	parsed, err := template.ParseFiles(templateFile)
	if err != nil {
		e.logger.Errorf("failed to parse %v error: %v", templateFile, err)
		return err
	}
	var body bytes.Buffer
	if errE := parsed.Execute(&body, templateMetaData{
		UserName:  userName,
		LoginMode: loginMode,
		TimeStamp: timeStamp,
		Env:       e.cfg.Env,
	}); errE != nil {
		e.logger.Errorf("failed to update dynamic data in html body, %v", errE)
		return errE
	}

	emailData := []Message{
		{
			From:    e.cfg.SenderEmail,
			To:      addresses,
			Subject: teamMailSubject,
			Body:    body.String(),
		},
	}
	if err := e.sendBatchEmails(ctx, emailData, e.endpoint); err != nil {
		e.logger.Errorf("error in sending notification mail to tasTeam: %v", err)
	}

	e.logger.Debugf("notification of new user signup on TAS send to team")
	return nil
}

func (e *emailNotificationManager) welcomeMail(ctx context.Context, user *core.GitUser) error {
	userAdress := map[string]string{user.Email: user.Username}
	addresses, err := e.validateAddresses(userAdress)
	if err != nil {
		e.logger.Errorf("error in validating email addresses: %v, error: %v", user.Email, err)
		return err
	}

	templateFile := welcomeTemplate
	parsed, err := template.ParseFiles(templateFile)
	if err != nil {
		e.logger.Errorf("failed to parse %v error: %v", templateFile, err)
		return err
	}

	var body bytes.Buffer
	if errE := parsed.Execute(&body, templateMetaData{
		UserName:     user.Username,
		DashboardURL: fmt.Sprintf("%s/%s/%s/dashboard", e.cfg.FrontendURL, user.GitProvider, user.Username),
		IndexPageURL: e.cfg.FrontendURL,
		ImageURL:     e.rocketImageURL,
	}); errE != nil {
		e.logger.Errorf("failed to update dynamic data in html body, %v", errE)
		return errE
	}

	emailData := []Message{
		{
			From:    e.cfg.SenderEmail,
			To:      addresses,
			Subject: fmt.Sprintf("Welcome %s! Get started with Test At Scale", user.Username),
			Body:    body.String(),
		},
	}
	if err := e.sendBatchEmails(ctx, emailData, e.endpoint); err != nil {
		e.logger.Errorf("error in sending welcome email to %s: %v", user.Email, err)
	}

	e.logger.Debugf("welcome email sent to %s", user.Username)
	return nil
}

func (e *emailNotificationManager) orgAdminsMail(ctx context.Context, newUser string,
	gitProvider core.SCMDriver, adminDetails []orgAdminDetail) error {
	if len(adminDetails) == 0 {
		return errs.New("admins details not found")
	}

	batchEmailsData := []Message{}
	templateFile := newUserInExistingOrgTemplate
	for _, orgAdmin := range adminDetails {
		orgName := orgAdmin.orgName
		adminName := orgAdmin.adminName
		adminEmail := orgAdmin.adminEmail

		// if an error occur in any step, just skip sending mail for that particular org
		address, err := e.validateAddresses(map[string]string{adminEmail: adminName})
		if err != nil {
			e.logger.Errorf("error in validating email addresses for org %s, email: %v, error: %v", orgName, adminEmail, err)
		} else {
			parsed, err := template.ParseFiles(templateFile)
			if err != nil {
				e.logger.Errorf("failed to parse %v error: %v", templateFile, err)
			} else {
				dashboardURL := fmt.Sprintf("%s/%s/%s/dashboard/", e.cfg.FrontendURL, gitProvider, orgName)
				var body bytes.Buffer
				if err := parsed.Execute(&body, templateMetaData{
					AdminUserName:   adminName,
					UserName:        newUser,
					ImageURL:        e.rocketImageURL,
					OrgName:         orgName,
					OrgDashboardURL: dashboardURL,
				}); err != nil {
					e.logger.Errorf("failed to update dynamic data in html body, for org %s, error: %v", orgName, err)
				} else {
					emailData := &Message{
						From:    e.cfg.SenderEmail,
						To:      address,
						Subject: newUserInExistingOrgSubject,
						Body:    body.String(),
					}
					batchEmailsData = append(batchEmailsData, *emailData)
				}
			}
		}
	}

	if err := e.sendBatchEmails(ctx, batchEmailsData, e.endpoint); err != nil {
		e.logger.Errorf("error in sending emails in batch to org admins: %v", err)
		return err
	}
	e.logger.Debugf("notification for new user signup on TAS in existing org send to respective org's admins")
	return nil
}

func (e *emailNotificationManager) OpenSourceUserNotification(ctx context.Context, user *core.OpenSourceUserInfo) error {
	if _, err := mail.ParseAddress(e.cfg.SenderEmail); err != nil {
		e.logger.Errorf("invalid sender email: %v", err)
		return err
	}

	addresses, err := e.validateAddresses(e.cfg.TasTeam)
	if err != nil {
		e.logger.Errorf("error in validating email addresses: %v, error: %v", e.cfg.TasTeam, err)
		return err
	}

	templateFile := openSourceUserTemplate
	parsed, err := template.ParseFiles(templateFile)
	if err != nil {
		e.logger.Errorf("failed to parse %v error: %v", templateFile, err)
		return err
	}

	var body bytes.Buffer
	if errE := parsed.Execute(&body, struct {
		Name        string
		EmailID     string
		RepoLink    string
		Description string
	}{
		Name:        fmt.Sprintf("%s %s", user.FirstName, user.LastName),
		EmailID:     user.EmailID,
		RepoLink:    user.Link,
		Description: user.Description,
	}); errE != nil {
		e.logger.Errorf("failed to update dynamic data in html body, %v", errE)
		return errE
	}

	emailData := []Message{
		{
			From:    e.cfg.SenderEmail,
			To:      addresses,
			Subject: openSourceUserSubject,
			Body:    body.String(),
		},
	}
	if err := e.sendBatchEmails(ctx, emailData, e.endpoint); err != nil {
		e.logger.Errorf("error in sending demo notification mail to tasTeam: %v", err)
	}

	e.logger.Debugf("demo signup notification mail send to team")
	return nil
}
