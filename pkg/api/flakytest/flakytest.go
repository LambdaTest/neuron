package flakytest

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

// HandleFlakyBuildDetails return details for a particular build for a repo
func HandleFlakyBuildDetails(
	flakyTestStore core.FlakyTestStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		buildID := c.Param("buildID")
		taskID := c.Query("taskID")
		statusFilter := c.Query("status")
		executionStatusFilter := c.Query("execution_status")
		searchFilter := c.Query("text")

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()
		tests, err := flakyTestStore.FindTests(ctx, buildID, taskID, statusFilter,
			executionStatusFilter, searchFilter, cd.Offset, cd.Limit+1)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Tests", "build_id"))
				return
			}
			logger.Errorf("error while finding tests for buildID %s orgID %s, repoName %s, %v",
				buildID, cd.OrgID, cd.RepoName, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		responseMetadata := new(core.ResponseMetadata)
		if len(tests) == cd.Limit+1 {
			responseMetadata.NextCursor = utils.EncodeOffset(cd.Offset + cd.Limit)
			tests = tests[:len(tests)-1]
		}
		c.JSON(http.StatusOK, gin.H{"tests": tests, "response_metadata": responseMetadata})
	}
}
