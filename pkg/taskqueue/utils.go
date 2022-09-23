package taskqueue

import (
	"context"
	"time"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"gopkg.in/guregu/null.v4/zero"
)

type taskQueueUtils struct {
	logger           lumber.Logger
	taskQueueStore   core.TaskQueueStore
	gitStatusService core.GitStatusService
	buildMonitor     core.BuildMonitor
}

// NewTaskQueueUtils returns new taskQueueUtils
func NewTaskQueueUtils(logger lumber.Logger,
	taskQueueStore core.TaskQueueStore,
	gitStatusService core.GitStatusService,
	buildMonitor core.BuildMonitor) core.TaskQueueUtils {
	return &taskQueueUtils{logger: logger, taskQueueStore: taskQueueStore, gitStatusService: gitStatusService, buildMonitor: buildMonitor}
}

func (t *taskQueueUtils) MarkTaskToStatus(task *core.Task, orgID string, status core.Status) {
	now := time.Now()
	taskObj := &core.Task{
		ID:      task.ID,
		Type:    task.Type,
		BuildID: task.BuildID,
		Status:  status,
		Remark:  task.Remark,
		EndTime: zero.TimeFrom(now),
		Updated: now}
	if err := t.taskQueueStore.UpdateTask(context.Background(), taskObj, orgID); err != nil {
		t.logger.Errorf("error while marking taskID  %s as %s, orgID %s, buildID %s, error %v",
			task.ID, status, orgID, task.BuildID, err)
	}
	// update the build depending on task
	if err := t.buildMonitor.FindAndUpdate(context.Background(), orgID, taskObj); err != nil {
		t.logger.Errorf("error while updating buildID %s in queue, error: %v", taskObj.BuildID, err)
		return
	}
}
