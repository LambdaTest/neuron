package organization

import (
	"context"
	"errors"
	"net/http"
	str "strconv"
	"time"

	apiutils "github.com/LambdaTest/neuron/pkg/api/utils"
	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/gin-gonic/gin"
)

// OrgRunnerInfo is the request struct of synapse runner
type OrgRunnerInfo struct {
	Org        string `json:"org"`
	RunnerType string `json:"runner_type"`
}

// TokenInfo is the request struct of tokrn creation
type TokenInfo struct {
	Org   string `json:"org"`
	Token string `json:"token"`
}

// HandleList lists the users organizations
func HandleList(orgStore core.OrganizationStore, logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, nil, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err.Error())
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		orgs, err := orgStore.FindUserOrgs(ctx, cd.UserID)

		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"message": "No orgs not found for given user."})
				return
			}
			logger.Errorf("failed to find orgs for user: %s, %v", cd.UserID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": errs.GenericErrorMessage})
			return
		}

		c.JSON(http.StatusOK, orgs)
	}
}

// HandleSync lists the users organizations after refetching it from scm
func HandleSync(userStore core.GitUserStore,
	orgService core.OrganizationService,
	orgStore core.OrganizationStore,
	tokenHandler core.GitTokenHandler,
	loginStore core.LoginStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, nil, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err.Error())
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		user, err := userStore.FindByID(ctx, cd.UserID)
		if err != nil {
			logger.Errorf("failed to fetch user info from db: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": errs.GenericErrorMessage})
			return
		}

		token, _, err := tokenHandler.GetToken(ctx, false, cd.GitProvider, "", "", user.Mask, user.ID)
		if err != nil {
			logger.Errorf("failed to fetch token for userID %s : %v", user.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": errs.GenericErrorMessage})
			return
		}

		user.Oauth = *token

		orgs, err := orgService.List(ctx, cd.GitProvider, user, token)
		if err != nil {
			logger.Errorf("failed to find git org list from scm: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": errs.GenericErrorMessage})
			return
		}

		if cd.GitProvider == core.DriverGithub {
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

		orgz, err := loginStore.UpdateOrgList(ctx, user, orgs)
		if err != nil {
			logger.Errorf("failed to find orgs for user: %s, %v", cd.UserID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": errs.GenericErrorMessage})
			return
		}

		c.JSON(http.StatusOK, orgz)
	}
}

// HandleUpdateConfigInfo updates the config info for an org
func HandleUpdateConfigInfo(orgStore core.OrganizationStore, userStore core.GitUserStore, logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var runnerInfo OrgRunnerInfo
		if err := c.ShouldBindJSON(&runnerInfo); err != nil {
			logger.Errorf("error while binding json %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}

		cd, statusCode, err := apiutils.ExtractAndValidateData(c, nil, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		_, orgID, err := userStore.FindByOrg(ctx, cd.UserID, runnerInfo.Org)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "org"))
				return
			}
			logger.Errorf("error while finding user for org %s, %v", cd.OrgName, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		secretKey := utils.GenerateUUID()

		if err := orgStore.UpdateConfigInfo(ctx, orgID, secretKey, runnerInfo.RunnerType); err != nil {
			logger.Errorf("failed to insert user info, error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": errs.GenericErrorMessage})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Info added Successfully.", "token": secretKey})
	}
}

// HandleUpdateSynapseToken updates the synapse token for an org
func HandleUpdateSynapseToken(orgStore core.OrganizationStore, userStore core.GitUserStore, logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenInfo TokenInfo
		if err := c.ShouldBindJSON(&tokenInfo); err != nil {
			logger.Errorf("error while binding json %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, nil, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		_, orgID, err := userStore.FindByOrg(ctx, cd.UserID, tokenInfo.Org)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "org"))
				return
			}
			logger.Errorf("error while finding user for org %s, %v", cd.OrgName, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		secretKey := utils.GenerateUUID()

		if err := orgStore.UpdateSynapseToken(ctx, orgID, secretKey, tokenInfo.Token); err != nil {
			logger.Errorf("failed to insert user info, error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": errs.GenericErrorMessage})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "New token added Successfully.", "token": secretKey})
	}
}

// HandleSynapseList returns list of synapse for an org
func HandleSynapseList(synapseStore core.SynapseStore, userStore core.GitUserStore, logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"org": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		_, orgID, err := userStore.FindByOrg(ctx, cd.UserID, cd.OrgName)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "org"))
				return
			}
			logger.Errorf("error while finding user for org %s, %v", cd.OrgName, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		synapseList, err := synapseStore.GetSynapseList(ctx, orgID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			logger.Errorf("error while finding synapse meta for org %s, %v", cd.OrgName, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"synapse_list": synapseList})
	}
}

// HandleSynapseTestConnection tests if synapse for an org is connected exist or not
func HandleSynapseTestConnection(synapseStore core.SynapseStore, userStore core.GitUserStore, logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"org": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		_, orgID, err := userStore.FindByOrg(ctx, cd.UserID, cd.OrgName)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "org"))
				return
			}
			logger.Errorf("error while finding user for org %s, %v", cd.OrgName, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		isSynapseExist, err := synapseStore.TestSynapseConnection(ctx, orgID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			logger.Errorf("error while testing connection of synapse for org %s, %v", cd.OrgName, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"synapse_connection": isSynapseExist})
	}
}

// HandleSynapsePatch updates is_active in db
func HandleSynapsePatch(synapseStore core.SynapseStore, userStore core.GitUserStore, logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"org": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		_, orgID, err := userStore.FindByOrg(ctx, cd.UserID, cd.OrgName)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "org"))
				return
			}
			logger.Errorf("error while finding user for org %s, %v", cd.OrgName, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		synapseID, ok := c.GetQuery("synapseID")
		if !ok {
			c.JSON(http.StatusBadRequest, errs.MissingInQueryErr("synapseID"))
			return
		}
		isActive, ok := c.GetQuery("is_active")
		if !ok {
			c.JSON(http.StatusBadRequest, errs.MissingInQueryErr("is_active"))
			return
		}
		active, err := str.ParseBool(isActive)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid type of is_active , it must by boolean type"})
		}
		if err := synapseStore.UpdateIsActiveSynapse(ctx, active, synapseID, orgID); err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"message": "No such synapseID present"})
				return
			}
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "successfully updated"})
	}
}

// HandleSynapseCount returns the count of connected and total register synapse
func HandleSynapseCount(synapseStore core.SynapseStore, userStore core.GitUserStore, logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"org": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		_, orgID, err := userStore.FindByOrg(ctx, cd.UserID, cd.OrgName)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "org"))
				return
			}
			logger.Errorf("error while finding user for org %s, %v", cd.OrgName, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		synapseCountList, err := synapseStore.CountSynapse(ctx, orgID)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "org"))
				return
			}
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.JSON(http.StatusOK, synapseCountList)
	}
}
