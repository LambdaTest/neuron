package buildmonitor

import (
	"context"
	"time"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
)

const schedulerTime = 15 * time.Minute

type scheduler struct {
	logger  lumber.Logger
	manager core.SchedulerManager
}

// NewScheduler returns a new scheduler to stop stale tasks and builds.
func NewScheduler(
	logger lumber.Logger,
	manager core.SchedulerManager,
) core.Scheduler {
	return &scheduler{
		logger:  logger,
		manager: manager,
	}
}

// Run will execute after every 15 mins to remove stale builds.
func (s *scheduler) Run(ctx context.Context) {
	s.logger.Infof("Starting scheduler to remove stale builds and tasks.")
	timer := time.NewTicker(schedulerTime)
	for {
		select {
		case <-ctx.Done():
			s.logger.Debugf("Closed scheduler to remove stale builds and tasks")
			return
		case <-timer.C:
			err := s.manager.StopStaleBuilds(ctx)
			if err != nil {
				s.logger.Errorf("failed to stop stale builds error %v", err)
			}
		}
	}
}
