package testsuite

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	apiutils "github.com/LambdaTest/neuron/pkg/api/utils"
	"github.com/LambdaTest/neuron/pkg/constants"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/gin-gonic/gin"
)

// HandleList return testssuites for a repo
func HandleList(
	testSuiteStore core.TestSuiteStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		suiteID := c.Param("suiteID")
		branchName := c.Query("branch")
		statusFilter := c.Query("status")
		searchText := c.Query("text")
		authorsNames := c.QueryArray("author")
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		suites, err := testSuiteStore.FindByRepo(ctx, cd.RepoName, cd.OrgID, suiteID, branchName,
			statusFilter, searchText, authorsNames, cd.Offset, cd.Limit+1)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Statuses", "suiteID"))
				return
			}
			logger.Errorf("error while finding statuses for suiteID %s orgID %s, repoID %s, %v",
				suiteID, cd.OrgID, cd.RepoID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		responseMetadata := new(core.ResponseMetadata)
		if len(suites) == cd.Limit+1 {
			responseMetadata.NextCursor = utils.EncodeOffset(cd.Offset + cd.Limit)
			suites = suites[:len(suites)-1]
		}
		c.JSON(http.StatusOK, gin.H{"test_suites": suites, "response_metadata": responseMetadata})
	}
}

// HandleDetails returns details of test suite for a repo
func HandleDetails(
	testSuiteStore core.TestSuiteStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {},
			"start_date": {}, "end_date": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		suiteID := c.Param("suiteID")
		branchName := c.Query("branch")
		statusFilter := c.Query("status")
		searchID := c.Query("id")

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		testSuites, err := testSuiteStore.FindExecution(ctx,
			suiteID,
			cd.RepoName,
			cd.OrgID,
			branchName,
			statusFilter,
			searchID,
			cd.StartDate,
			cd.EndDate,
			cd.Offset,
			cd.Limit+1)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Suite details", "suiteID"))
				return
			}
			logger.Errorf("error while finding details of suiteID %s for orgID %s, repoID %s, %v",
				suiteID, cd.OrgID, cd.RepoID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		responseMetadata := new(core.ResponseMetadata)

		if len(testSuites) == cd.Limit+1 {
			responseMetadata.NextCursor = utils.EncodeOffset(cd.Offset + cd.Limit)
			testSuites = testSuites[:len(testSuites)-1]
		}
		c.JSON(http.StatusOK, gin.H{"test_suite_results": testSuites, "response_metadata": responseMetadata})
	}
}

// HandleStatusData returns status details of test suite  for a repo
func HandleStatusData(
	testSuiteStore core.TestSuiteStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {},
			"start_date": {}, "end_date": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		suiteID := c.Param("suiteID")
		branchName := c.Query("branch")

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		testsuites, err := testSuiteStore.FindStatus(ctx, suiteID, cd.RepoName, cd.OrgID, branchName, cd.StartDate, cd.EndDate)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Statuses", "suiteID"))
				return
			}
			logger.Errorf("error while finding statuses for suiteID %s, orgID %s, repoID %s, %v",
				suiteID, cd.OrgID, cd.RepoID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.JSON(http.StatusOK, testsuites)
	}
}

// HandleExecMetricsBlob returns metrics of test suite  for a repo
func HandleExecMetricsBlob(
	testExecutionService core.TestExecutionService,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c,
			map[string]struct{}{"repo": {}, "org": {}, "task_id": {}, "build_id": {}, "exec_id": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		testSuiteMetricsPath := fmt.Sprintf("test_suite/%s/%s/%s/%s/%s", cd.OrgID, cd.RepoID, cd.BuildID, cd.TaskID,
			constants.DefaultMetricsFileName)
		metrics, err := testExecutionService.FetchMetrics(ctx, testSuiteMetricsPath, cd.ExecID)
		if err != nil {
			if errors.Is(err, errs.ErrNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Metrics", "suiteID"))
				return
			}
			logger.Errorf("error while finding metrics for taskID %s in buildID %s  orgID %s, repoID %s, error: %v",
				cd.BuildID, cd.TaskID, cd.OrgID, cd.RepoID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.JSON(http.StatusOK, metrics)
	}
}

// HandleMeta return testssuites for a repo
func HandleMeta(
	testSuiteStore core.TestSuiteStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		buildID := c.Query("build_id")
		commitID := c.Query("commit_id")
		branchName := c.Query("branch")
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		suites, err := testSuiteStore.FindMeta(ctx, cd.RepoName, cd.OrgID, buildID, commitID, branchName)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Statuses", "suiteID"))
				return
			}
			logger.Errorf("error while finding meta for orgID %s, repoID %s, %v",
				cd.OrgID, cd.RepoID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.JSON(http.StatusOK, gin.H{"test_suites": suites})
	}
}
