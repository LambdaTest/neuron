package synapse

import (
	"context"
	"errors"
	"net/http"

	apiutils "github.com/LambdaTest/neuron/pkg/api/utils"
	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/gin-gonic/gin"
)

// HandleList Lists available synapse for an organization
func HandleList(synapseStore core.SynapseStore, userStore core.GitUserStore, logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctxData, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"org": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()
		_, orgID, err := userStore.FindByOrg(ctx, ctxData.UserID, ctxData.OrgName)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "org"))
				return
			}
			logger.Errorf("error while finding user for org %s, %v", ctxData.OrgName, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		synapseList, err := synapseStore.ListSynapseMeta(ctx, orgID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.JSON(http.StatusOK, gin.H{"synapse_list": synapseList})
	}
}
