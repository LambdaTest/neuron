package build

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	apiutils "github.com/LambdaTest/neuron/pkg/api/utils"
	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/drone/go-scm/scm"
	"github.com/gin-gonic/gin"
)

// ReBuildRequest stores rebuild request
type ReBuildRequest struct {
	Org     string `json:"org"`
	Repo    string `json:"repo"`
	BuildID string `json:"build_id"`
}

// HandleList return builds for a repo
func HandleList(
	buildStore core.BuildStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		buildID := c.Param("buildID")
		branchName := c.Query("branch")
		statusFilter := c.Query("status")
		searchID := c.Query("id")
		authorsNames := c.QueryArray("author")
		tags := c.QueryArray("tag")
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

		builds, err := buildStore.FindByRepo(ctx, cd.RepoName, cd.OrgID, buildID, branchName, statusFilter,
			searchID, authorsNames, tags, nextStartTime, cd.Limit+1)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Builds", "user"))
				return
			}
			logger.Errorf("error while finding builds for orgID %s, repoName %s, %v", cd.OrgID, cd.RepoName, err)
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

// HandleFindTimeSaved finds the time saved by not running unimpacted tests.
func HandleFindTimeSaved(
	buildMonitor core.BuildMonitor,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		buildID := c.Param("buildID")
		commitID := c.Query("sha")
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		timeSaved, err := buildMonitor.FindTimeSavedData(ctx, buildID, commitID, cd.RepoID)
		if err != nil {
			logger.Errorf("error while finding time saved data for orgID: %s, repoID: %s, buildID: %s, commitID: %s, error: %v",
				cd.OrgID, cd.RepoID, buildID, commitID, err)
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Tests", "buildID and commitID"))
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"message": errs.GenericErrorMessage})
			return
		}

		c.JSON(http.StatusOK, timeSaved)
	}
}

// HandleBuildDetails return details for a particular build for a repo
func HandleBuildDetails(
	buildStore core.BuildStore,
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

		tests, err := buildStore.FindTests(ctx, buildID, cd.RepoName, cd.OrgID, cd.Offset, cd.Limit+1)
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

// HandleFetchMetrics returns details of test metrics for a repo
func HandleFetchMetrics(
	repoStore core.RepoStore,
	testExecutionService core.TestExecutionService,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c,
			map[string]struct{}{"repo": {}, "org": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		buildID := c.Param("buildID")
		taskID := c.Param("taskID")

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		buildMetricsPath := fmt.Sprintf("test/%s/%s/%s/%s/%s", cd.OrgID, cd.RepoID, buildID, taskID, constants.DefaultMetricsFileName)
		metrics, err := testExecutionService.FetchMetrics(ctx, buildMetricsPath, "")
		if err != nil {
			if errors.Is(err, errs.ErrNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Metrics", "buildID"))
				return
			}
			logger.Errorf("error while finding metrics for taskID %s in buildID %s, orgID %s, repoID %s, %v",
				taskID, buildID, cd.OrgID, cd.RepoID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.JSON(http.StatusOK, metrics)
	}
}

// HandleListTasks fetches the list of all tasks which are there in a particular build
func HandleListTasks(taskStore core.TaskStore, logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}, "build_id": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		tasks, err := taskStore.FetchTask(ctx, cd.BuildID, cd.Offset, cd.Limit+1)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Tasks", "repository"))
				return
			}
			logger.Errorf("error while finding tasks for buildID %s, repoID %s, orgID %s, %v", cd.BuildID, cd.RepoID, cd.OrgID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		responseMetadata := new(core.ResponseMetadata)
		// set the element as next_cursor value
		if len(tasks) == cd.Limit+1 {
			responseMetadata.NextCursor = utils.EncodeOffset(cd.Offset + cd.Limit)
			// remove last element to return len==limit
			tasks = tasks[:len(tasks)-1]
		}
		c.JSON(http.StatusOK, gin.H{"tasks": tasks, "response_metadata": responseMetadata})
	}
}

// HandleFetchLogs returns details of logs for a build
func HandleFetchLogs(
	azureClient core.AzureBlob,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c,
			map[string]struct{}{"org": {}, "task_id": {}, "logs_tag": {}}, false)
		if err != nil {
			c.JSON(statusCode, err)
			return
		}
		buildID := c.Param("buildID")
		blobPath := fmt.Sprintf("%s/%s/%s/%s.log", cd.OrgID, buildID, cd.TaskID, cd.LogsTag)
		logsPath, err := azureClient.GenerateSasURL(c, blobPath, core.LogsContainer)
		if err != nil {
			logger.Errorf("failed to generate SAS URL for  logPath %s, buildID %s, error %v", blobPath, buildID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.JSON(http.StatusOK, gin.H{"path": azureClient.ReplaceWithCDN(logsPath)})
	}
}

// HandleRebuild re-builds a particular buildID
func HandleRebuild(
	buildService core.BuildService,
	taskStore core.TaskStore,
	logger lumber.Logger,
	repoStore core.RepoStore,
	userStore core.GitUserStore,
	buildStore core.BuildStore,
	gitEventStore core.GitEventStore,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqBody := &ReBuildRequest{}
		if err := c.ShouldBindJSON(reqBody); err != nil {
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

		_, orgID, err := userStore.FindByOrg(ctx, cd.UserID, reqBody.Org)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "org"))
				return
			}
			logger.Errorf("error while finding user for org %s, %v", reqBody.Org, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		repo, err := repoStore.Find(ctx, orgID, reqBody.Repo)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "repo"))
				return
			}
			logger.Errorf("failed to find repoID for repo: %s error: %v", reqBody.Repo, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		build, err := buildStore.FindByBuildID(ctx, reqBody.BuildID)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"message": errs.EntityNotFoundErr("buildID", "repo")})
				return
			}
			logger.Errorf("error while finding build for orgID %s, repoID %s, buildID %s, error %v",
				orgID, repo.ID, reqBody.BuildID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		gitEvent, err := gitEventStore.FindByBuildID(ctx, reqBody.BuildID)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"message": errs.EntityNotFoundErr("git_event", "repo")})
				return
			}
			logger.Errorf("error while finding git event for orgID %s, repoID %s, buildID %s, error %v",
				orgID, repo.ID, reqBody.BuildID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		err = handlePushEventRebuild(gitEvent, build.CommitID, logger)
		if err != nil {
			logger.Errorf("failed to handle push event rebuild orgID %s, repoID %s, buildID %s, error %v",
				orgID, repo.ID, reqBody.BuildID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		parserResponse, err := buildService.ParseGitEvent(ctx, gitEvent, repo, build.Tag)
		if err != nil {
			logger.Errorf("failed to parse webhook for rebuild orgID %s, repoID %s, buildID %s, error %v",
				orgID, repo.ID, reqBody.BuildID, err)
			c.JSON(createErrorResponse(err))
			return
		}
		httpResponse, err := buildService.CreateBuild(ctx, parserResponse, gitEvent.GitProviderHandle, false)
		if err != nil {
			logger.Errorf("failed to create build for rebuild orgID %s, repoID %s, buildID %s, error %v",
				orgID, repo.ID, reqBody.BuildID, err)
			c.JSON(httpResponse, errs.GenericErrorMessage)
			return
		}
		c.Status(httpResponse)
	}
}

func createErrorResponse(err error) (int, error) {
	if errors.Is(err, errs.ErrPingEvent) {
		return http.StatusNoContent, nil
	}
	if errors.Is(err, errs.ErrUnknownEvent) {
		return http.StatusNotFound, err
	}
	if errors.Is(err, errs.ErrQuery) || errors.Is(err, errs.ErrMarshalJSON) || errors.Is(err, errs.ErrAzureUpload) {
		return http.StatusInternalServerError, errs.GenericErrorMessage
	}
	return http.StatusBadRequest, err
}

func handlePushEventRebuild(gitEvent *core.GitEvent, commitID string, logger lumber.Logger) error {
	// if the original event was a push event, filter and build only the required commit
	if gitEvent.EventName == core.EventPush {
		var v *scm.PushHook
		err := json.Unmarshal(gitEvent.EventPayload, &v)
		if err != nil {
			logger.Errorf("could not parse event payload to push hook, err: %v", err)
			return errors.New("failed to parse git event associated with commit")
		}
		for i := range v.Commits {
			commit := v.Commits[i]
			if commit.Sha == commitID {
				if i != 0 {
					v.Before = v.Commits[i-1].Sha
				}
				if i != len(v.Commits)-1 {
					v.After = v.Commits[i+1].Sha
				}
				v.Commit = commit
				v.Commits = []scm.Commit{commit}
				break
			}
		}
		gitEvent.EventPayload, err = json.Marshal(v)
		if err != nil {
			logger.Errorf("could not parse event payload to push hook, err: %v", err)
			return errors.New("failed to parse git event associated with commit")
		}
		gitEvent.CommitID = commitID
	}
	return nil
}

// HandleListMeta returns build Meta for a repo
func HandleListMeta(
	buildStore core.BuildStore,
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

		builds, err := buildStore.FindBuildMeta(ctx, cd.RepoName, cd.OrgID, branchName)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Builds", "user"))
				return
			}
			logger.Errorf("error while finding build meta for orgID %s, repoName %s, %v", cd.OrgID, cd.RepoName, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		c.JSON(http.StatusOK, gin.H{"builds_meta": builds})
	}
}

// HandleFindDiff return commit diff for a repo
func HandleFindDiff(
	buildStore core.BuildStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		buildID := c.Query("build_id")
		gitProvider := c.Query("git_provider")
		commitDiff, err := buildStore.FindCommitDiff(ctx, cd.OrgName, cd.RepoName, gitProvider, buildID)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Diff", "jobs"))
				return
			}
			logger.Errorf("error while finding diff for this job %s, repoName %s, %v", buildID, cd.RepoName, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		c.JSON(http.StatusOK, gin.H{"commit_diff_url": commitDiff})
	}
}

// HandleAbort aborts the queued or running build
func HandleAbort(buildAbortService core.BuildAbortService,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		buildID := c.Query("buildID")
		if buildID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"message": "missing buildID in parameters"})
			return
		}
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		logger.Debugf("request received to abort build with buildID: %s, orgID: %s, repoID: %s", buildID, cd.OrgID, cd.RepoID)
		msg, err := buildAbortService.AbortBuild(ctx, buildID, cd.RepoID, cd.OrgID)
		if err != nil {
			logger.Errorf("failed to abort build for buildID: %s, repoID: %s, orgID: %s, error: %v", buildID, cd.RepoID, cd.OrgID, err)
			if errors.Is(err, errs.EntityNotFoundErr("build", "buildID and repository")) || errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("build", "buildID and repository"))
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"message": errs.GenericErrorMessage})
			return
		}

		if msg == core.JobCompletedMsg {
			c.JSON(http.StatusOK, gin.H{"message": core.JobCompletedMsg})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "aborting job inititated"})
	}
}
