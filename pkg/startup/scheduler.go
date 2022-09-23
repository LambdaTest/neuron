package startup

import (
	"context"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
)

type scheduler struct {
	logger           lumber.Logger
	taskQueueStore   core.TaskQueueStore
	taskQueueManager core.TaskQueueManager
	eventQueueStore  core.EventQueueStore
}

// TODO: Remove once BG deployed and tested on production

// NewScheduler returns a new scheduler to run stuck tasks
func NewScheduler(
	logger lumber.Logger,
	taskQueueStore core.TaskQueueStore,
	taskQueueManager core.TaskQueueManager,
	eventQueueStore core.EventQueueStore) core.Scheduler {
	return &scheduler{
		logger:           logger,
		taskQueueStore:   taskQueueStore,
		taskQueueManager: taskQueueManager,
		eventQueueStore:  eventQueueStore,
	}
}

// will execute on startup to start running stuck tasks
func (s *scheduler) Run(ctx context.Context) {
	s.logger.Infof("Starting scheduler on startup to run stuck tasks in queue.")
	// find the organizations having tasks with status ready in queue
	orgs, err := s.eventQueueStore.FindOrgs(ctx, core.Ready, core.TaskEvent)
	if err != nil {
		s.logger.Errorf("failed to finds tasks with status ready in queue, error %v", err)
	}
	for i := range orgs {
		go func(orgID string) {
			if err = s.taskQueueManager.DequeueTasks(orgID); err != nil {
				s.logger.Errorf("error while dequeueing tasks from queue for orgID %s, error: %v", orgID, err)
			}
		}(orgs[i])
	}
}
