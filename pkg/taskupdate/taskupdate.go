package taskupdate

import (
	"context"
	"time"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"golang.org/x/sync/errgroup"
	"gopkg.in/guregu/null.v4/zero"
)

type taskUpdate struct {
	taskQueueStore   core.TaskQueueStore
	buildStore       core.BuildStore
	taskStore        core.TaskStore
	buildMonitor     core.BuildMonitor
	taskQueueManager core.TaskQueueManager
	runner           core.K8sRunner
	logger           lumber.Logger
}

// New returns a new TaskUpdateManager
func New(
	taskQueueStore core.TaskQueueStore,
	buildStore core.BuildStore,
	taskStore core.TaskStore,
	buildMonitor core.BuildMonitor,
	taskQueueManager core.TaskQueueManager,
	runner core.K8sRunner,
	logger lumber.Logger,
) core.TaskUpdateManager {
	return &taskUpdate{
		taskQueueStore:   taskQueueStore,
		buildStore:       buildStore,
		taskStore:        taskStore,
		buildMonitor:     buildMonitor,
		taskQueueManager: taskQueueManager,
		runner:           runner,
		logger:           logger,
	}
}

func (u *taskUpdate) TaskUpdate(
	ctx context.Context,
	task *core.Task,
	orgID string,
) error {
	if err := u.taskQueueStore.UpdateTask(ctx, task, orgID); err != nil {
		u.logger.Errorf("error while updating task queue store, task: %s, org: %s, error: %v", task.ID, orgID, err)
		return err
	}
	if errF := u.buildMonitor.FindAndUpdate(ctx, orgID, task); errF != nil {
		u.logger.Errorf("error while updating build %s in queue, error: %v", task.BuildID, errF)
		return errF
	}
	if utils.TaskFinished(task.Status) {
		go func(taskID, buildID, orgID string) {
			if err := u.runner.WaitForPodTermination(context.Background(),
				utils.GetNucleusPodName(taskID),
				utils.GetRunnerNamespaceFromOrgID(orgID)); err != nil {
				u.logger.Errorf("error while waiting for pod termination, error: %v", err)
				// not returning as can give error due to non zero exit code and pod is deleted
			}
			if err := u.taskQueueManager.DequeueTasks(orgID); err != nil {
				u.logger.Errorf("error while dequeueing tasks from queue for orgID %s, buildID %s, taskID %s  error: %v",
					orgID, buildID, taskID, err)
			}
		}(task.ID, task.BuildID, orgID)
	}
	return nil
}

func (u *taskUpdate) UpdateAllTasksForBuild(ctx context.Context, remark, buildID, orgID string) error {
	status := core.TaskInitiating
	tasks, err := u.taskStore.FetchTaskHavingStatus(ctx, buildID, status)
	if err != nil {
		u.logger.Errorf("failed to find tasks for buildID: %s, error: %v", buildID, err)
	}

	g, errCtx := errgroup.WithContext(ctx)
	for _, task := range tasks {
		func(task *core.Task) {
			g.Go(func() error {
				now := time.Now()
				task.Status = core.TaskAborted
				task.Updated = now
				task.EndTime = zero.TimeFrom(now)
				task.Remark = zero.StringFrom(remark)
				if errT := u.TaskUpdate(errCtx, task, orgID); err != nil {
					u.logger.Errorf("failed to update task for taskID: %s, buildID: %s, orgID: %s, error: %v", task.ID, buildID, orgID, errT)
					return errT
				}
				return nil
			})
		}(task)
	}
	if err = g.Wait(); err != nil {
		u.logger.Errorf("error while marking aborting tasks for buildID: %s, error: %v", buildID, err)
		return err
	}
	return nil
}
