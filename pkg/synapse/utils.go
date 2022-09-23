package synapse

import (
	"context"
	"fmt"

	errs "github.com/LambdaTest/neuron/pkg/errors"
	"gopkg.in/guregu/null.v4/zero"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
)

// Specs denotes system specification
type Specs struct {
	CPU float32
	RAM int64
}

// TierOpts is const map which map each tier to specs
var TierOpts = map[core.Tier]Specs{
	core.Internal: {CPU: 0.5, RAM: 384},
	core.XSmall:   {CPU: 1, RAM: 2000},
	core.Small:    {CPU: 2, RAM: 4000},
	core.Medium:   {CPU: 4, RAM: 8000},
	core.Large:    {CPU: 8, RAM: 16000},
	core.XLarge:   {CPU: 16, RAM: 32000},
}

// GetResources function to get dummy resources based on pod type
func GetResources(tierOpts core.Tier) Specs {
	if val, ok := TierOpts[tierOpts]; ok {
		return val
	}
	return Specs{CPU: 0, RAM: 0}
}

// CheckSynpaseMinRequirement checks if synapse has enough resources
func CheckSynpaseMinRequirement(cpu float32, ram int64) bool {
	if cpu < TierOpts[core.Small].CPU || ram < TierOpts[core.Small].RAM {
		return false
	}
	return true
}

func removeAt(slice []core.SynapseJobInfo, index int) []core.SynapseJobInfo {
	if index < 0 {
		return nil
	}
	if index >= len(slice) {
		return nil
	}

	newSlice := make([]core.SynapseJobInfo, 0)
	newSlice = append(newSlice, slice[:index]...)
	if index != len(slice)-1 {
		newSlice = append(newSlice, slice[index+1:]...)
	}

	return newSlice
}

// incase of nucleus crash mark task as error and schedule the next task
func (s *synapseManager) handleFailedTestExecJob(ctx context.Context,
	jobInfo *core.JobInfo,
	repo *core.Repository,
	taskStore core.TaskStore,
	taskQueueUtils core.TaskQueueUtils,
	taskQueueManager core.TaskQueueManager,
	remarks string,
	updateRemarks bool,
	logger lumber.Logger) {
	job := core.Job{
		ID:     jobInfo.JobID,
		TaskID: jobInfo.ID,
		OrgID:  repo.OrgID,
		Status: core.Failed,
	}
	task, err := taskStore.Find(ctx, job.TaskID)
	if err != nil {
		logger.Errorf("failed to find task details in database for orgID %s, taskID %s, error: %v",
			job.OrgID, job.TaskID, err)
		return
	}
	if remarks == "" {
		task.Remark = zero.StringFrom(errs.GenericUserFacingBEErrRemark.Error())
	} else {
		task.Remark = zero.StringFrom(remarks)
	}
	// only update the remarks at first time
	if updateRemarks {
		if err := s.buildStore.UpdateBuildCache(ctx, jobInfo.BuildID, map[string]interface{}{core.BuildFailedRemarks: remarks},
			false); err != nil {
			s.logger.Errorf("error while updating %s: %s  in buildCache for taskID %s buildID %s orgID %s , err %v",
				core.BuildFailedRemarks, remarks, jobInfo.ID, jobInfo.BuildID, job.OrgID, err)
		}
	}

	logger.Errorf("JobID %s orgID %s, taskID %s, buildID %s failed, marking the job as failed", jobInfo.JobID,
		job.OrgID, task.ID, task.BuildID)
	taskQueueUtils.MarkTaskToStatus(task, job.OrgID, core.TaskError)
	go func(orgID string) {
		if err := taskQueueManager.DequeueTasks(orgID); err != nil {
			logger.Errorf("error while dequeueing tasks from queue for orgID %s, buildID %s taskID %s error: %v", orgID,
				task.BuildID, task.ID, err)
		}
	}(job.OrgID)
}

func getSynapseKey(orgID, synapseID string) string {
	return fmt.Sprintf("synapse-%s-%s", orgID, synapseID)
}
