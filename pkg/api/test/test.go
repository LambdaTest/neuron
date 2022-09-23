package test

import (
	"context"
	"errors"

	"net/http"

	apiutils "github.com/LambdaTest/neuron/pkg/api/utils"
	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/utils"

	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/gin-gonic/gin"
)

// HandleList return tests for a repo
func HandleList(
	testStore core.TestStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		testID := c.Param("testID")
		branchName := c.Query("branch")
		statusFilter := c.Query("status")
		searchText := c.Query("text")
		authorsNames := c.QueryArray("author")

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		// By setting the limit to one more than the count requested by the client,
		// we’ll know we’re at the last page when the number of rows returned is less than count
		tests, err := testStore.FindByRepo(ctx, cd.RepoName, cd.OrgID, testID, branchName, statusFilter,
			searchText, authorsNames, cd.Offset, cd.Limit+1)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Tests", "repository"))
				return
			}
			logger.Errorf("error while finding tests for orgID %s, repoID %s, %v", cd.OrgID, cd.RepoID, err)
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

// HandleTestMeta returns the meta for all the tests
func HandleTestMeta(
	testStore core.TestStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		branchName := c.Query("branch")
		commitID := c.Query("commit_id")
		buildID := c.Query("build_id")

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		tests, err := testStore.FindTestMeta(ctx, cd.RepoName, cd.OrgID, buildID, commitID, branchName)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Tests", "repository"))
				return
			}
			logger.Errorf("error while finding tests meta for orgID %s, repoID %s, %v", cd.OrgID, cd.RepoID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.JSON(http.StatusOK, gin.H{"tests": tests})
	}
}
