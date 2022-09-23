package login

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/utils"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/drone/go-login/login"
	"github.com/gin-gonic/gin"
)

//TODO: ignore self-hosted login

// Handler manages oauth login handler
func Handler(
	loginProvider core.GitLoginProvider,
	scmProvider core.SCMProvider,
	userService core.GitUserService,
	orgService core.OrganizationService,
	session core.Session,
	userStore core.GitUserStore,
	loginStore core.LoginStore,
	orgStore core.OrganizationStore,
	frontendURL string,
	githubAppName string,
	emailNotificationManager core.EmailNotificationManager,
	tokenHandler core.GitTokenHandler,
	githubApp core.GithubApp,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		pathParam := c.Param("scmdriver")

		// Installtion id for github app
		githubInstallationID := c.Query("installation_id")

		driver := core.SCMDriver(pathParam)
		if err := driver.VerifyDriver(); err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, err)
			return
		}

		if driver == core.DriverGithub {
			state := c.Query("state")
			c.Request.AddCookie(&http.Cookie{Name: "_oauth_state_", Value: state})
		}

		loginHandler, err := loginProvider.Get(driver)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"message": "Failed to find login client."})
			return
		}

		handler := handler(
			driver,
			githubInstallationID,
			userService,
			orgService,
			session,
			userStore,
			loginStore,
			orgStore,
			frontendURL,
			githubAppName,
			emailNotificationManager,
			tokenHandler,
			githubApp,
			logger)
		loginHandler.Handler(handler).ServeHTTP(c.Writer, c.Request)
	}
}

// handler manages oauth login handler
func handler(
	driver core.SCMDriver,
	githubInstallationID string,
	userService core.GitUserService,
	orgService core.OrganizationService,
	session core.Session,
	userStore core.GitUserStore,
	loginStore core.LoginStore,
	orgStore core.OrganizationStore,
	frontendURL,
	githubAppName string,
	emailNotificationManager core.EmailNotificationManager,
	tokenHandler core.GitTokenHandler,
	githubApp core.GithubApp,
	logger lumber.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		err := login.ErrorFrom(ctx)
		if err != nil {
			logger.Errorf("cannot authenticate user: %v", err)
			http.Redirect(w, r, frontendURL, http.StatusSeeOther)
			return
		}

		// The authorization token is passed from the
		// login middleware in the context.
		tok := login.TokenFrom(ctx)

		src, err := userService.Find(ctx, driver, tok)
		if err != nil {
			logger.Errorf("failed to find git user from scm: %v", err)
			http.Error(w, errs.GenericErrorMessage.Error(), http.StatusInternalServerError)
			return
		}
		userExists := true

		user, err := userStore.Find(ctx, src.Username, src.GitProvider)
		if err != nil {
			if !errors.Is(err, errs.ErrRowsNotFound) {
				logger.Errorf("failed to find git user in database: %v", err)
				http.Error(w, errs.GenericErrorMessage.Error(), http.StatusInternalServerError)
				return
			}
			userExists = false
		}

		if !userExists {
			user = src
		} else {
			// set login credentials
			user.Oauth = src.Oauth
			user.Email = src.Email
		}

		orgs, installedOrg, err := getOrgDetails(ctx, user, driver, orgService, orgStore, githubApp, githubInstallationID, logger)
		if err != nil {
			logger.Errorf("Error while fetching org details for user %+v , err %s", user, err)
			http.Error(w, errs.GenericErrorMessage.Error(), http.StatusInternalServerError)
			return
		}
		if len(orgs) == 0 {
			http.Redirect(w, r, fmt.Sprintf("https://github.com/apps/%s/installations/new", githubAppName), http.StatusSeeOther)
			return
		}

		if err = loginStore.Create(ctx, user, userExists, orgs...); err != nil {
			logger.Errorf("error while running queries in transaction for login, error: %v", err)
			http.Error(w, errs.GenericErrorMessage.Error(), http.StatusInternalServerError)
			return
		}

		if githubInstallationID != "" {
			err = updateInstallationToken(ctx, orgStore, installedOrg, tokenHandler, &user.Oauth, logger)
			if err != nil {
				logger.Errorf("error while updating instllation token, error: %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		cookie, err := session.CreateToken(&core.UserData{UserID: user.ID, GitProvider: user.GitProvider, UserName: user.Username})
		if err != nil {
			logger.Errorf("error while creating JWT token, error %v", err)
			http.Error(w, errs.GenericErrorMessage.Error(), http.StatusInternalServerError)
			return
		}

		if !userExists {
			go func() {
				err := emailNotificationManager.SendMailOnNewUserSignUp(context.Background(), user, frontendURL, orgs)
				// not returning error because email failure should not disturb login flow
				if err != nil {
					logger.Errorf("failed to send email after signup error %v", err)
				}
			}()
		}
		http.SetCookie(w, cookie)

		http.Redirect(w, r, fmt.Sprintf("%s/%s/select-org/", frontendURL, driver.String()), http.StatusSeeOther)
	}
}

//nolint:gocritic
func getOrgDetails(ctx context.Context,
	user *core.GitUser,
	driver core.SCMDriver,
	orgService core.OrganizationService,
	orgStore core.OrganizationStore,
	githubApp core.GithubApp,
	installationID string,
	logger lumber.Logger) ([]*core.Organization, *core.Organization, error) {
	orgs, err := orgService.List(ctx, driver, user, &user.Oauth)
	if err != nil {
		logger.Errorf("failed to find git org list from scm: %v", err)
		return orgs, nil, err
	}

	if driver == core.DriverGithub {
		now := time.Now()
		userOrg := &core.Organization{ID: utils.GenerateUUID(),
			Name:        user.Username,
			Avatar:      user.Avatar,
			GitProvider: user.GitProvider,
			Created:     now,
			Updated:     now}

		_, err = orgStore.Find(ctx, userOrg)
		if err == nil {
			orgs = append(orgs, userOrg)
		}
	}

	var installedOrg *core.Organization

	// If installationID is present that means login was called through a github installation
	if installationID != "" {
		orgName, installationToken, err := githubApp.GetInstallation(installationID)
		if err != nil {
			logger.Errorf("Error while fetching installation details of org : %s", err)
			return orgs, nil, err
		}

		user.Oauth.InstallationID = installationToken.InstallationID
		user.Oauth.InstallationExpiry = installationToken.InstallationExpiry
		user.Oauth.InstallationToken = installationToken.InstallationToken

		installedOrg, err = orgService.GetOrganization(ctx, driver, user, &user.Oauth, orgName)
		if err != nil {
			logger.Errorf("failed to find git org from scm: %v", err)
			return orgs, nil, err
		}
		orgs = append(orgs, installedOrg)
	}

	return orgs, installedOrg, nil
}

func updateInstallationToken(ctx context.Context,
	orgStore core.OrganizationStore,
	installedOrg *core.Organization,
	tokenHandler core.GitTokenHandler,
	installationToken *core.Token,
	logger lumber.Logger) error {
	orgID, err := orgStore.Find(ctx, installedOrg)
	if err != nil {
		logger.Errorf("error while getting organization ID, err: %s", err)
		return err
	}

	err = tokenHandler.UpdateInstallationToken(core.DriverGithub,
		utils.GetRunnerNamespaceFromOrgID(orgID),
		installedOrg.Name, installationToken)
	if err != nil {
		logger.Errorf("error while updating instllation token in vault, err: %s", err)
		return err
	}
	return nil
}
