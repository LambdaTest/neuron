package commit

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	apiutils "github.com/LambdaTest/neuron/pkg/api/utils"
	"github.com/LambdaTest/neuron/pkg/constants"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/utils"
	"golang.org/x/sync/errgroup"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/gin-gonic/gin"
)

// HandleLists lists the all the commits of a repo
func HandleLists(
	commitStore core.GitCommitStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}, "git_provider": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		commitID := c.Param("sha")
		branchName := c.Query("branch")
		statusFilter := c.Query("status")
		searchText := c.Query("text")
		authorsNames := c.QueryArray("author")
		buildID := c.Query("buildID")

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		commits, err := commitStore.FindByRepo(ctx, cd.RepoName, cd.OrgID, commitID, buildID, branchName, cd.OrgName,
			cd.GitProviderType, statusFilter, searchText,
			authorsNames, cd.Offset, cd.Limit+1)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Commits", "repository"))
				return
			}
			logger.Errorf("failed to find commit for repoID %s, orgID %s, commitID %s, error: %v",
				cd.RepoID, cd.OrgID, commitID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		responseMetadata := new(core.ResponseMetadata)
		// set the element as next_cursor value
		if len(commits) == cd.Limit+1 {
			responseMetadata.NextCursor = utils.EncodeOffset(cd.Offset + cd.Limit)
			// remove last element to return len==limit
			commits = commits[:len(commits)-1]
		}
		c.JSON(http.StatusOK, gin.H{"commits": commits, "response_metadata": responseMetadata})
	}
}

// HandleListBuilds lists the builds of a commit
func HandleListBuilds(
	buildStore core.BuildStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		commitID := c.Param("sha")
		branchName := c.Query("branch")

		var nextStartTime time.Time
		if cd.NextCursor != "" {
			unixTime, perr := strconv.ParseInt(cd.NextCursor, constants.Base10, constants.BitSize64)
			if perr != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, errs.InvalidQueryErr("next_cursor"))
				return
			}
			nextStartTime = time.Unix(unixTime, 0)
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		builds, err := buildStore.FindByCommit(ctx, commitID, cd.RepoName, cd.OrgID, branchName, nextStartTime, cd.Limit+1)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Builds", "repo"))
				return
			}
			logger.Errorf("failed to find commit for repoID %s, orgID %s, commitID %s, error: %v",
				cd.RepoID, cd.OrgID, commitID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		responseMetadata := new(core.ResponseMetadata)
		if len(builds) == cd.Limit+1 {
			responseMetadata.NextCursor = utils.EncodeCursor(strconv.FormatInt(builds[len(builds)-1].Created.Unix(), constants.Base10))
			builds = builds[:len(builds)-1]
		}
		c.JSON(http.StatusOK, gin.H{"builds": builds, "response_metadata": responseMetadata})
	}
}

// HandleListImpactedTests lists the impacted tests of a build details for a commit
func HandleListImpactedTests(
	commitStore core.GitCommitStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		commitID := c.Param("sha")
		buildID := c.Param("buildID")
		searchText := c.Query("text")
		status := c.Query("status")
		branchName := c.Query("branch")
		taskID := c.Query("task_id")

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		tests, err := commitStore.FindImpactedTests(ctx, commitID, buildID, taskID, cd.RepoName, cd.OrgID, status,
			searchText, branchName, cd.Offset, cd.Limit+1)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Tests", "commit and buildID"))
				return
			}
			logger.Errorf("failed to find test details for commitID %s, buildID %s, repoID %s, orgID %s, error: %v",
				commitID, buildID, cd.RepoID, cd.OrgID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		responseMetadata := new(core.ResponseMetadata)
		if len(tests) == cd.Limit+1 {
			responseMetadata.NextCursor = utils.EncodeOffset(cd.Offset + cd.Limit)
			tests = tests[:len(tests)-1]
		}
		c.JSON(http.StatusOK, gin.H{"impacted_tests": tests, "response_metadata": responseMetadata})
	}
}

// HandleListUnimpactedTests lists the un-impacted tests for a particular build of that commit
func HandleListUnimpactedTests(
	testStore core.TestStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctxData, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		commitID := c.Param("sha")
		buildID := c.Param("buildID")

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		tests, err := testStore.FindUnimpactedTests(ctx, commitID, buildID, ctxData.RepoName, ctxData.OrgID, ctxData.Offset, ctxData.Limit+1)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Unimpacted tests", "commit and buildID"))
				return
			}
			logger.Errorf("failed to find unimpacted test details for commitID %s, buildID %s, repoID %s, orgID %s, error: %v",
				commitID, buildID, ctxData.RepoName, ctxData.OrgID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		responseMetadata := new(core.ResponseMetadata)
		if len(tests) == ctxData.Limit+1 {
			responseMetadata.NextCursor = utils.EncodeOffset(ctxData.Offset + ctxData.Limit)
			tests = tests[:len(tests)-1]
		}
		c.JSON(http.StatusOK, gin.H{"unimpacted_tests": tests, "response_metadata": responseMetadata})
	}
}

// HandleListCommitsBuild lists the all the commits that are there in a particular build
func HandleListCommitsBuild(
	commitStore core.GitCommitStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		buildID := c.Param("buildID")

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()
		commits, err := commitStore.FindByBuild(ctx, cd.RepoName, cd.OrgID, buildID, cd.Offset, cd.Limit+1)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Commits", "buildID"))
				return
			}
			logger.Errorf("failed to find commit for repoID %s, orgID %s, error: %v", cd.RepoID, cd.OrgID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		responseMetadata := new(core.ResponseMetadata)
		if len(commits) == cd.Limit+1 {
			responseMetadata.NextCursor = utils.EncodeOffset(cd.Offset + cd.Limit)
			commits = commits[:len(commits)-1]
		}
		c.JSON(http.StatusOK, gin.H{"commits": commits, "response_metadata": responseMetadata})
	}
}

// HandleListTestsByCommit lists the impacted tests corresponds to a build for a particular commit
func HandleListTestsByCommit(
	commitStore core.GitCommitStore,
	testExecutionStore core.TestExecutionStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		commitID := c.Param("sha")
		buildID := c.Param("buildID")
		searchText := c.Query("text")
		status := c.Query("status")
		taskID := c.Query("task_id")
		branchName := c.Query("branch")
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()
		var tests []*core.TestExecution
		g, errCtx := errgroup.WithContext(ctx)
		g.Go(func() error {
			listtests, cerr := commitStore.FindImpactedTests(errCtx, commitID, buildID, taskID, cd.RepoName, cd.OrgID, status, searchText,
				branchName, cd.Offset, cd.Limit+1)
			if cerr != nil {
				return cerr
			}
			tests = listtests
			return nil
		})
		var timeAllTests int
		g.Go(func() error {
			_, totalTime, terr := testExecutionStore.FindExecutionTimeImpactedTests(errCtx, commitID, buildID, cd.RepoID)
			if terr != nil {
				return terr
			}
			timeAllTests = totalTime
			return nil
		})
		if err = g.Wait(); err != nil {
			if errors.Is(err, ctx.Err()) {
				logger.Errorf("context canceled while fetching details from database for orgID %s, repoID %s, buildID %s, error %v",
					cd.OrgID, cd.RepoID, buildID, err)
				c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			} else {
				if errors.Is(err, errs.ErrRowsNotFound) {
					c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Tests", "user"))
					return
				}
				logger.Errorf("error while fetching info from database for orgID %s, repoID %s, buildID %s, error %v",
					cd.OrgID, cd.RepoID, buildID, err)
				c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			}
			return
		}

		responseMetadata := new(core.ResponseMetadata)
		if len(tests) == cd.Limit+1 {
			responseMetadata.NextCursor = utils.EncodeOffset(cd.Offset + cd.Limit)
			tests = tests[:len(tests)-1]
		}
		c.JSON(http.StatusOK, gin.H{"impacted_tests": tests, "total_time": timeAllTests, "response_metadata": responseMetadata})
	}
}

// HandleListMeta lists the meta of all the commits of a repo
func HandleListMeta(
	commitStore core.GitCommitStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		branchName := c.Query("branch")

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		commits, err := commitStore.FindCommitMeta(ctx, cd.RepoName, cd.OrgID, branchName)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Commits", "repository"))
				return
			}
			logger.Errorf("failed to find commit meta for repoID %s, orgID %s, error: %v",
				cd.RepoID, cd.OrgID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		c.JSON(http.StatusOK, gin.H{"commits": commits})
	}
}
