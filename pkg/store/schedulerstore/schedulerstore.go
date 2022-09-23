package schedulerstore

import (
	"context"
	"time"

	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/jmoiron/sqlx"
)

const (
	maxRetries = 3
	delay      = 250 * time.Millisecond
	maxJitter  = 100 * time.Millisecond
	errMsg     = "failed to perform stale build transaction"
)

type manager struct {
	db              core.DB
	orgStore        core.OrganizationStore
	buildStore      core.BuildStore
	taskStore       core.TaskStore
	eventQueueStore core.EventQueueStore
	logger          lumber.Logger
}

// New returns a new scheduler manager to stop stale tasks and builds.
func New(
	db core.DB,
	orgStore core.OrganizationStore,
	buildStore core.BuildStore,
	taskStore core.TaskStore,
	eventQueueStore core.EventQueueStore,
	logger lumber.Logger,
) core.SchedulerManager {
	return &manager{
		db:              db,
		orgStore:        orgStore,
		logger:          logger,
		buildStore:      buildStore,
		taskStore:       taskStore,
		eventQueueStore: eventQueueStore,
	}
}

// StopStaleBuilds marks stale builds as error
func (m *manager) StopStaleBuilds(ctx context.Context) error {
	m.logger.Debugf("stopping stale builds")
	dbErr := m.db.ExecuteTransactionWithRetry(ctx, maxRetries, delay, maxJitter, errMsg,
		func(tx *sqlx.Tx) error {
			staleEvents, err := m.eventQueueStore.FetchStaleEventCountInTx(ctx, tx, constants.BuildTimeout)
			if err != nil {
				m.logger.Errorf("failed to find stale events %v", err)
				return err
			}
			buildCount, err := m.buildStore.StopStaleBuildsInTx(ctx, tx, constants.BuildTimeout)
			if err != nil {
				m.logger.Errorf("failed to update stale build, error %v", err)
				return err
			}
			if buildCount > 0 {
				m.logger.Infof("stopped %d stale builds in db", buildCount)
			}

			taskCount, err := m.taskStore.StopStaleTasksInTx(ctx, tx, constants.BuildTimeout)
			if err != nil {
				m.logger.Errorf("failed to update stale tasks, error %v", err)
				return err
			}
			if taskCount > 0 {
				m.logger.Infof("stopped %d stale tasks in db", taskCount)
			}

			eventQueueJobsCount, err := m.eventQueueStore.StopStaleEventsInTx(ctx, tx, constants.BuildTimeout)
			if err != nil {
				m.logger.Errorf("failed to update stale events, error %v", err)
				return err
			}
			if eventQueueJobsCount > 0 {
				m.logger.Infof("stopped %d stale events in db", eventQueueJobsCount)
			}

			if len(staleEvents) > 0 {
				for _, event := range staleEvents {
					err = m.orgStore.UpdateKeysInCache(ctx, event.OrgID,
						map[string]int64{"running_tasks": -event.ProcessingTasks, "queued_tasks": -event.ReadyTasks})
					if err != nil {
						m.logger.Errorf("failed to update concurrency details for orgID %s, error: %v", event.OrgID, err)
						return err
					}
				}
			}
			return nil
		})
	if dbErr != nil {
		return dbErr
	}
	return nil
}
