package task

import (
	"context"
	"net/http"
	"time"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/gin-gonic/gin"
	"gopkg.in/guregu/null.v4/zero"
)

type taskInput struct {
	TaskID      string         `json:"task_id"`
	Status      core.Status    `json:"status"`
	Remark      string         `json:"remark"`
	RepoSlug    string         `json:"repo_slug"`
	RepoLink    string         `json:"repo_link"`
	RepoID      string         `json:"repo_id"`
	OrgID       string         `json:"org_id"`
	Type        core.TaskType  `json:"type"`
	GitProvider core.SCMDriver `json:"git_provider"`
	BuildID     string         `json:"build_id"`
	StartTime   time.Time      `json:"start_time"`
	EndTime     time.Time      `json:"end_time,omitempty"`
}

// HandleUpdate updates the status of the task.
func HandleUpdate(
	buildStore core.BuildStore,
	taskUpdateManager core.TaskUpdateManager,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		taskDetails := new(taskInput)
		if err := c.ShouldBindJSON(taskDetails); err != nil {
			logger.Errorf("error while binding json %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()
		// TODO: add valdation of payload from DB.
		if err := validateRequestBody(taskDetails); err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}
		// check if build is still active
		err := buildStore.CheckBuildCache(ctx, taskDetails.BuildID)
		if err != nil {
			logger.Errorf("failed to find build: %s, task: %s details in cache error: %v", taskDetails.BuildID,
				taskDetails.TaskID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		t := &core.Task{
			ID:        taskDetails.TaskID,
			BuildID:   taskDetails.BuildID,
			RepoID:    taskDetails.RepoID,
			Status:    taskDetails.Status,
			Type:      taskDetails.Type,
			Updated:   time.Now(),
			Remark:    zero.StringFrom(taskDetails.Remark),
			StartTime: zero.TimeFrom(taskDetails.StartTime),
			EndTime:   zero.TimeFrom(taskDetails.EndTime),
		}
		if err := taskUpdateManager.TaskUpdate(ctx, t, taskDetails.OrgID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		c.Data(http.StatusOK, gin.MIMEPlain, []byte(http.StatusText(http.StatusOK)))
	}
}

func validateRequestBody(req *taskInput) error {
	if req.BuildID == "" {
		return errs.MissingInReqErr("buildID")
	}
	if req.RepoID == "" {
		return errs.MissingInReqErr("repoID")
	}
	if req.TaskID == "" {
		return errs.MissingInReqErr("taskID")
	}
	if req.OrgID == "" {
		return errs.MissingInReqErr("orgID")
	}
	if req.Type == "" {
		return errs.MissingInReqErr("type")
	}
	if req.Status == "" {
		return errs.MissingInReqErr("status")
	}
	return nil
}
