package credits

import (
	"context"
	"errors"
	"net/http"

	apiutils "github.com/LambdaTest/neuron/pkg/api/utils"
	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/gin-gonic/gin"
)

const (
	invalidEntityErrMessage = "Invalid entity requested. Should be 'orgid' or 'user'"
)

// HandleFind fetches credits-usage (in a paginated fashion) for a user or an org
func HandleFind(userStore core.GitUserStore,
	creditsUsageStore core.CreditsUsageStore,
	logger lumber.Logger,
	forEntity string) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"start_date": {}, "end_date": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		var usages []*core.CreditsUsage
		if forEntity == "user" {
			username := c.Param("username")
			usages, err = creditsUsageStore.FindByOrgOrUser(ctx, username, "", cd.StartDate, cd.EndDate, cd.Offset, cd.Limit+1)
		} else if forEntity == "org" {
			orgName := c.Query("org")
			_, orgID, oerr := userStore.FindByOrg(ctx, cd.UserID, orgName)
			if oerr != nil {
				if errors.Is(oerr, errs.ErrRowsNotFound) {
					c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "org"))
					return
				}
				logger.Errorf("error while finding user for org %s, %v", cd.OrgName, oerr)
				c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
				return
			}
			usages, err = creditsUsageStore.FindByOrgOrUser(ctx, "", orgID, cd.StartDate, cd.EndDate, cd.Offset, cd.Limit+1)
		} else {
			logger.Errorf("Invalid entity requested")
			c.JSON(http.StatusBadRequest, gin.H{"message": invalidEntityErrMessage})
			return
		}

		if err != nil {
			logger.Errorf("Error in fetching credits usage for %v entity, err %v", forEntity, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		responseMetadata := new(core.ResponseMetadata)
		if len(usages) == cd.Limit+1 {
			responseMetadata.NextCursor = utils.EncodeOffset(cd.Offset + cd.Limit)
			// remove last element to return len==limit
			usages = usages[:len(usages)-1]
		}
		c.JSON(http.StatusOK, gin.H{
			"data":              usages,
			"response_metadata": responseMetadata,
		})
	}
}

// HandleFindTotalConsumed fetches total consumed credits for an org with given timeline
func HandleFindTotalConsumed(creditsUsageStore core.CreditsUsageStore, logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{
			"start_date": {},
			"end_date":   {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		orgID := c.Query("org_id")
		if orgID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, errs.MissingInQueryErr("org_id"))
			return
		}

		totalConsumedCredits, err := creditsUsageStore.FindTotalConsumedCredits(ctx, orgID, cd.StartDate, cd.EndDate)
		if err != nil {
			logger.Errorf("Error in fetching total consumed credits for orgID %s, err %s", orgID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"totalConsumedCredits": totalConsumedCredits,
		})
	}
}
