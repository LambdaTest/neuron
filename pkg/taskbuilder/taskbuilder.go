package taskbuilder

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/gin-gonic/gin"
	"gopkg.in/guregu/null.v4/zero"
)

type taskBuilder struct {
	logger           lumber.Logger
	taskStore        core.TaskStore
	taskQueueManager core.TaskQueueManager
	buildStore       core.BuildStore
	azureClient      core.AzureBlob
}

func New(taskStore core.TaskStore,
	taskQueueManager core.TaskQueueManager,
	buildStore core.BuildStore,
	azureClient core.AzureBlob,
	logger lumber.Logger) core.TaskBuilder {
	return &taskBuilder{
		taskStore:        taskStore,
		taskQueueManager: taskQueueManager,
		buildStore:       buildStore,
		azureClient:      azureClient,
		logger:           logger,
	}
}

// getLocatorPayloadAddress uploads locators and return its payload address
func (t *taskBuilder) getLocatorPayloadAddress(ctx context.Context, locatorConfig *core.InputLocatorConfig,
	orgID, repoID, buildID, taskID string) (string, error) {
	payloadBytes, err := json.Marshal(locatorConfig)
	if err != nil {
		return "", err
	}
	payloadAddress, err := t.azureClient.UploadBytes(ctx, fmt.Sprintf("%s/%s/build/%s/task/%s", orgID, repoID, buildID, taskID),
		payloadBytes, core.PayloadContainer, gin.MIMEJSON)
	return payloadAddress, err
}

// CreateTask creates the task  object
func (t *taskBuilder) CreateTask(ctx context.Context,
	taskID, buildID, orgID, repoID, subModule string,
	taskType core.TaskType,
	locatorConfig *core.InputLocatorConfig,
	tier core.Tier) (*core.Task, error) {
	var locatorAddress string
	var err error
	if (taskType == core.ExecutionTask || taskType == core.FlakyTask) && len(locatorConfig.Locators) > 0 {
		locatorAddress, err = t.getLocatorPayloadAddress(context.Background(), locatorConfig, orgID, repoID, buildID, taskID)
		if err != nil {
			t.logger.Errorf("failed to upload blob to azure storage for buildID %s, orgID %s, repoID %s, error: %v",
				buildID, orgID, repoID, err)
			return nil, err
		}
	}
	buildCache, err := t.buildStore.GetBuildCache(ctx, buildID)
	if err != nil {
		t.logger.Errorf("error finding build cache for buildID %s, error %v", buildID, err)
		return nil, err
	}

	return &core.Task{
		ID:             taskID,
		BuildID:        buildID,
		RepoID:         repoID,
		Type:           taskType,
		Tier:           tier,
		Status:         core.TaskInitiating,
		TestLocators:   zero.StringFrom(locatorAddress),
		ContainerImage: buildCache.ContainerImage,
		SubModule:      subModule,
	}, nil
}

// CreateJob creates the job object
func (t *taskBuilder) CreateJob(taskID, orgID string, status core.QueueStatus) *core.Job {
	return &core.Job{
		ID:      utils.GenerateUUID(),
		TaskID:  taskID,
		Status:  status,
		OrgID:   orgID,
		Created: time.Now(),
		Updated: time.Now(),
	}
}

// EnqueueTaskAndJob creates the entry and in db and enqueue the job in queue
func (t *taskBuilder) EnqueueTaskAndJob(ctx context.Context,
	buildID, orgID string,
	tasks []*core.Task, jobs []*core.Job,
	taskType core.TaskType) error {
	// cache will be updated in case of execution task as of now
	if taskType == core.ExecutionTask {
		if err := t.buildStore.UpdateExecTaskCount(ctx, buildID, len(tasks)); err != nil {
			t.logger.Errorf("error updating exec task count for orgID %s buildID %s, error: %v",
				orgID, buildID, err)
			return err
		}
	}
	if err := t.taskStore.Create(ctx, tasks...); err != nil {
		t.logger.Errorf("failed insert tasks for orgID %s, buildID %s, error %v",
			orgID, buildID, err)
		return err
	}
	// only create task but skip the dequeue as they will be dequeued in /task callback API.
	if err := t.taskQueueManager.EnqueueTasks(orgID, buildID, jobs...); err != nil {
		t.logger.Errorf("failed insert task queue store for orgID %s, buildID %s, error %v",
			orgID, buildID, err)
		if buildErr := t.taskStore.UpdateByBuildID(ctx, core.TaskError,
			taskType, errs.GenericUserFacingBEErrRemark.Error(), buildID); buildErr != nil {
			t.logger.Errorf("failed mark all tasks as failed for orgID %s, buildID %s, error %v", orgID, buildID, buildErr)
		}
	}
	return nil
}
