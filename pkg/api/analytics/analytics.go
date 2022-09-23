package analytics

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

// HandleListTestStatuses return test statuses for a repo
func HandleListTestStatuses(
	testStore core.TestStore,
	logger lumber.Logger,
	isJobs bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c,
			map[string]struct{}{"repo": {}, "org": {}, "start_date": {}, "end_date": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		searchText := c.Query("text")
		branchName := c.Query("branch")
		status := c.Query("status")
		tag := c.Query("tag")

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		// By setting the limit to one more than the count requested by the client,
		// we’ll know we’re at the last page when the number of rows returned is less than count
		tests, err := testStore.FindRepoTestStatuses(ctx, cd.RepoName, cd.OrgID, branchName, status, tag, searchText,
			cd.NextCursor, cd.StartDate, cd.EndDate, cd.Limit+1, isJobs)
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
		// set the element as next_cursor value
		if len(tests) == cd.Limit+1 {
			responseMetadata.NextCursor = utils.EncodeCursor(tests[len(tests)-1].ID)
			// remove last element to return len==limit
			tests = tests[:len(tests)-1]
		}
		c.JSON(http.StatusOK, gin.H{"tests": tests, "response_metadata": responseMetadata})
	}
}

// HandleListBuildStatuses return build statuses for a repo
func HandleListBuildStatuses(
	buildStore core.BuildStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctxData, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {},
			"start_date": {}, "end_date": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		branchName := c.Query("branch")
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		builds, err := buildStore.FindBuildStatus(ctx, ctxData.RepoName, ctxData.OrgID, branchName, ctxData.StartDate, ctxData.EndDate)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Builds", "repository"))
				return
			}
			logger.Errorf("error while finding builds for orgID %s, repoID %s, %v", ctxData.OrgID,
				ctxData.RepoID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		c.JSON(http.StatusOK, gin.H{"builds": builds})
	}
}

// HandleListCommitStatuses return commit statuses for a repo
func HandleListCommitStatuses(
	commitStore core.GitCommitStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctxData, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		branchName := c.Query("branch")
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		testsData, err := commitStore.FindCommitStatus(ctx, ctxData.RepoName, ctxData.OrgID, branchName, ctxData.Limit)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Commits", "repository"))
				return
			}
			logger.Errorf("error while finding tests for orgID %s, repoID %s, %v", ctxData.OrgID, ctxData.RepoID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		c.JSON(http.StatusOK, gin.H{"commits": testsData})
	}
}

// HandleListTestStatusesSlowest return test statuses for a repo
func HandleListTestStatusesSlowest(
	testStore core.TestStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {},
			"start_date": {}, "end_date": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()
		branchName := c.Query("branch")
		tests, err := testStore.FindRepoTestStatusesSlowest(ctx, cd.RepoName, cd.OrgID, branchName, cd.StartDate, cd.EndDate, cd.Limit)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Tests", "repository"))
				return
			}
			logger.Errorf("error while finding tests for orgID %s, repoID %s, %v", cd.OrgID, cd.RepoID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.JSON(http.StatusOK, gin.H{"tests": tests})
	}
}

// HandleListTestStatusesFailed return test statuses for a repo
func HandleListTestStatusesFailed(
	testExecutionStore core.TestExecutionStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctxData, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {},
			"org": {}, "start_date": {}, "end_date": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		branchName := c.Query("branch")
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		testsFailed, err := testExecutionStore.FindRepoTestStatusesFailed(ctx, ctxData.RepoName, ctxData.OrgID, branchName,
			ctxData.StartDate, ctxData.EndDate, ctxData.Limit)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Tests", "repository"))
				return
			}
			logger.Errorf("error while finding tests for orgID %s, repoID %s, %v", ctxData.OrgID, ctxData.RepoID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.JSON(http.StatusOK, gin.H{"tests": testsFailed})
	}
}

// HandleListBuildStatusesFailed return build statuses for a repo
func HandleListBuildStatusesFailed(
	buildStore core.BuildStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()
		ctxData, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {},
			"org": {}, "start_date": {}, "end_date": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		branchName := c.Query("branch")

		builds, err := buildStore.FindBuildStatusFailed(ctx, ctxData.RepoName, ctxData.OrgID, branchName,
			ctxData.StartDate, ctxData.EndDate, ctxData.Limit)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Builds", "repository"))
				return
			}
			logger.Errorf("error while finding builds for orgID %s, repoID %s, %v",
				ctxData.OrgID, ctxData.RepoID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		c.JSON(http.StatusOK, gin.H{"builds": builds})
	}
}

// HandleListTestDataCommits return test data with respect to commit id
func HandleListTestDataCommits(
	testStore core.TestStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()
		branchName := c.Query("branch")
		ctxData, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {},
			"org": {}, "start_date": {}, "end_date": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		testData, err := testStore.FindTestData(ctx, ctxData.RepoName, ctxData.OrgID, branchName,
			ctxData.StartDate, ctxData.EndDate)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Builds", "repository"))
				return
			}
			logger.Errorf("error while finding tests for orgID %s, repoID %s, %v",
				ctxData.OrgID, ctxData.RepoID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.JSON(http.StatusOK, gin.H{"test_data": testData})
	}
}

// HandleListFlakyTestsJobs return flaky test data with respect to build id
func HandleListFlakyTestsJobs(
	flakytestStore core.FlakyTestStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctxData, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {},
			"org": {}, "start_date": {}, "end_date": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()
		branchName := c.Query("branch")

		flakyTestData, err := flakytestStore.FindTestsInJob(ctx, ctxData.RepoName, ctxData.OrgID, branchName,
			ctxData.StartDate, ctxData.EndDate)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Builds", "repository"))
				return
			}
			logger.Errorf("error while finding flaky tests data for orgID %s, repoID %s, %v",
				ctxData.OrgID, ctxData.RepoID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.JSON(http.StatusOK, gin.H{"flakytests_data": flakyTestData})
	}
}

// HandleListFlakyTests return flaky test list data
func HandleListFlakyTests(
	flakytestStore core.FlakyTestStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctxData, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {},
			"org": {}, "start_date": {}, "end_date": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()
		branchName := c.Query("branch")

		flakyTestData, err := flakytestStore.ListFlakyTests(ctx, ctxData.RepoName, ctxData.OrgID, branchName,
			ctxData.StartDate, ctxData.EndDate, ctxData.Offset, ctxData.Limit+1)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Builds", "repository"))
				return
			}
			logger.Errorf("error while finding flaky tests data for orgID %s, repoID %s, %v",
				ctxData.OrgID, ctxData.RepoID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		responseMetadata := new(core.ResponseMetadata)
		if len(flakyTestData) == ctxData.Limit+1 {
			responseMetadata.NextCursor = utils.EncodeOffset(ctxData.Offset + ctxData.Limit)
			flakyTestData = flakyTestData[:len(flakyTestData)-1]
		}

		c.JSON(http.StatusOK, gin.H{"flakytests_data": flakyTestData, "response_metadata": responseMetadata})
	}
}

// HandleListTestStatusesIrregular returns irregular tests for a repo
func HandleListTestStatusesIrregular(
	testStore core.TestStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {},
			"start_date": {}, "end_date": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()
		branchName := c.Query("branch")
		irrTests, err := testStore.FindIrregularTests(ctx, cd.RepoID, branchName, cd.StartDate, cd.EndDate, cd.Limit)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Tests", "repository"))
				return
			}
			logger.Errorf("error while finding tests for orgID %s, repoID %s, %v", cd.OrgID, cd.RepoID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.JSON(http.StatusOK, gin.H{"irregular_tests": irrTests})
	}
}

// HandleListMonthWiseTest is a handler for finding monthwise total tests in a repo between start-date and end-date
func HandleListMonthWiseTest(
	testStore core.TestStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()
		branchName := c.Query("branch")
		ctxData, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {},
			"org": {}, "start_date": {}, "end_date": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		testData, err := testStore.FindMonthWiseNetTests(ctx, ctxData.RepoID, branchName, ctxData.StartDate, ctxData.EndDate)
		if err != nil {
			logger.Errorf("error while finding tests for orgID %s, repoID %s, %v",
				ctxData.OrgID, ctxData.RepoID, err)
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Commit", "repository"))
				return
			}
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		if len(testData) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"test_data": "no commit found in last 3 months"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"test_data": testData})
	}
}

// HandleListBuildMTTFandMTTR returns MTTF and MTTR of postmerge-jobs of a repo for a branch
func HandleListBuildMTTFandMTTR(
	buildStore core.BuildStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctxData, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {},
			"start_date": {}, "end_date": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		branchName := c.Query("branch")
		if branchName == "" {
			logger.Errorf("branch name not found in query params")
			c.JSON(http.StatusBadRequest, errs.MissingInQueryErr("branch"))
			return
		}
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		mttfAndMttr, err := buildStore.FindMttfAndMttr(ctx, ctxData.RepoID, branchName, ctxData.StartDate, ctxData.EndDate)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Builds", "repository"))
				return
			}
			logger.Errorf("error while finding builds for orgID %s, repoID %s, %v", ctxData.OrgID,
				ctxData.RepoID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.JSON(http.StatusOK, gin.H{"mttf_mttr": mttfAndMttr})
	}
}

// HandleListBuildWiseTimeSaved is handler for returning build wise time saved data in a given start and end date
func HandleListBuildWiseTimeSaved(
	buildStore core.BuildStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {},
			"start_date": {}, "end_date": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		branchName := c.Query("branch")
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		buildWiseTimeSaved, err := buildStore.FindBuildWiseTimeSaved(ctx, cd.RepoID, branchName, cd.StartDate, cd.EndDate, cd.Limit+1, cd.Offset)
		if err != nil {
			logger.Errorf("error while fetching buildWiseTimeSaved data from datastore for orgID: %s, repoID: %s, branch: %s, error: %v",
				cd.OrgID, cd.RepoID, branchName, err)
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("builds", "repository"))
				return
			}
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		responseMetaData := new(core.ResponseMetadata)
		if len(buildWiseTimeSaved) == cd.Limit+1 {
			responseMetaData.NextCursor = utils.EncodeOffset(cd.Offset + cd.Limit)
			buildWiseTimeSaved = buildWiseTimeSaved[:len(buildWiseTimeSaved)-1]
		}
		c.JSON(http.StatusOK, gin.H{"build_wise_time_saved": buildWiseTimeSaved, "response_metadata": responseMetaData})
	}
}

// HandleListTestsWithJobsStatuses return jobs statuses for a repo
func HandleListTestsWithJobsStatuses(
	buildStore core.BuildStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cdData, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {},
			"org": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		branchName := c.Query("branch")

		buildData, err := buildStore.FindJobsWithTestsStatus(ctx, cdData.RepoName, cdData.OrgID, branchName, cdData.Limit)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Builds", "repository"))
				return
			}
			logger.Errorf("error while finding tests for orgID %s, repoID %s, %v", cdData.OrgID, cdData.RepoID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		c.JSON(http.StatusOK, gin.H{"builds": buildData})
	}
}

// HandleListTestAdded returns the added test count in duration
func HandleListTestAdded(
	testStore core.TestStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()
		branchName := c.Query("branch")
		ctxData, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {},
			"org": {}, "start_date": {}, "end_date": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		addedTests, err := testStore.FindAddedTests(ctx, ctxData.RepoID, branchName,
			ctxData.StartDate, ctxData.EndDate)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Commit", "repository"))
				return
			}
			logger.Errorf("error while finding tests for orgID %s, repoID %s, %v",
				ctxData.OrgID, ctxData.RepoID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.JSON(http.StatusOK, gin.H{"added_tests_count": addedTests})
	}
}
