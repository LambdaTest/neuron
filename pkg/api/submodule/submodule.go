package submodule

import (
	"context"
	"net/http"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/gin-gonic/gin"
)

type subModuleList struct {
	BuildID        string `json:"buildID" binding:"required"`
	TotalSubModule int    `json:"totalSubModule" binding:"required,min=1"`
}

// HandleCreate creates set totalSubmodule to mentioned value in build cache
func HandleCreate(
	buildStore core.BuildStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqBody := &subModuleList{}
		if err := c.ShouldBindJSON(reqBody); err != nil {
			logger.Errorf("error while binding json, error: %v", err)
			c.JSON(http.StatusBadRequest, errs.ValidationErr(err))
			return
		}
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()
		updateMap := map[string]interface{}{
			core.TotalSubModule: reqBody.TotalSubModule,
		}
		if err := buildStore.UpdateBuildCache(ctx, reqBody.BuildID, updateMap, false); err != nil {
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.JSON(http.StatusOK, nil)
	}
}
