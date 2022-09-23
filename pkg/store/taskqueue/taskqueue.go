package taskqueue

import (
	"context"
	"errors"
	"time"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"gopkg.in/guregu/null.v4/zero"
)

const (
	maxRetries = 3
	delay      = 250 * time.Millisecond
	maxJitter  = 100 * time.Millisecond
	errMsg     = "failed to perform taskqueue transaction"
)

type taskQueueStore struct {
	db               core.DB
	licenseStore     core.LicenseStore
	taskStore        core.TaskStore
	orgStore         core.OrganizationStore
	creditUsageStore core.CreditsUsageStore
	buildStore       core.BuildStore
	eventQueueStore  core.EventQueueStore
	redisDB          core.RedisDB
	logger           lumber.Logger
}

// New returns a new TaskQueueStore.
func New(db core.DB, redisDB core.RedisDB,
	taskStore core.TaskStore,
	buildStore core.BuildStore,
	orgStore core.OrganizationStore,
	licenseStore core.LicenseStore,
	creditUsageStore core.CreditsUsageStore,
	eventQueueStore core.EventQueueStore,
	logger lumber.Logger) core.TaskQueueStore {
	return &taskQueueStore{
		db:               db,
		redisDB:          redisDB,
		logger:           logger,
		orgStore:         orgStore,
		licenseStore:     licenseStore,
		taskStore:        taskStore,
		buildStore:       buildStore,
		creditUsageStore: creditUsageStore,
		eventQueueStore:  eventQueueStore,
	}
}

func (t *taskQueueStore) Create(ctx context.Context,
	orgID,
	buildID string,
	tasks ...*core.Job) error {
	return t.db.ExecuteTransactionWithRetry(ctx, maxRetries, delay, maxJitter, errMsg, func(tx *sqlx.Tx) error {
		events := make([]*core.QueueableEvent, 0, len(tasks))
		for _, task := range tasks {
			events = append(events, task.ToQueueableEvent())
		}
		if err := t.eventQueueStore.CreateInTx(ctx, tx, events...); err != nil {
			t.logger.Errorf("failed to update tasks in task_queue table for orgID %s, buildID %s, error: %v", orgID, buildID, err)
			return err
		}
		orgKey := utils.GetOrgHashKey(orgID)
		// check if task concurrency field  exists in org hash
		existsCmd := t.redisDB.Client().HExists(ctx, orgKey, "running_tasks")
		exists, err := existsCmd.Result()
		if err != nil {
			t.logger.Errorf("failed to check if key exists for orgID %s, buildID %s, error: %v", orgID, buildID, err)
			return err
		}
		if !exists {
			ready, processing, err := t.eventQueueStore.FetchStatusCountInTx(ctx, tx, orgID, core.TaskEvent, true)
			if err != nil {
				t.logger.Errorf("failed to fetch status count from db for orgID %s, buildID %s, error: %v", orgID, buildID, err)
				return err
			}
			license, err := t.licenseStore.FindInTx(ctx, tx, orgID)
			if err != nil {
				t.logger.Errorf("failed to fetch license from db for orgID %s, buildID %s, error: %v", orgID, buildID, err)
				return err
			}
			if err := t.redisDB.Client().HSet(context.Background(), orgKey,
				map[string]interface{}{"queued_tasks": ready,
					"running_tasks":     processing,
					"total_concurrency": license.Concurrency}).Err(); err != nil {
				t.logger.Errorf("failed to update conncurrency details in redis for orgID %s, buildID %s, error: %v", orgID, buildID, err)
				return err
			}
			return nil
		}
		t.logger.Debugf("Incrementing queued_tasks for key: %s by %d", orgKey, len(tasks))
		if err := t.redisDB.Client().HIncrBy(context.Background(), orgKey, "queued_tasks", int64(len(tasks))).Err(); err != nil {
			t.logger.Errorf("failed to update conncurrency details in redis for orgID %s, buildID %s, error: %v", orgID, buildID, err)
			return err
		}
		return nil
	})
}

func (t *taskQueueStore) FindAndUpdateTasks(ctx context.Context, orgID string, limit int) ([]*core.Job, error) {
	jobs := make([]*core.Job, 0, limit)
	return jobs, t.db.ExecuteTransactionWithRetry(ctx, maxRetries, delay, maxJitter, errMsg, func(tx *sqlx.Tx) error {
		events, err := t.eventQueueStore.FindByOrgIDAndStatusInTx(ctx, tx, orgID, core.Ready, core.TaskEvent, true, limit)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				t.logger.Debugf("no tasks found in task_queue for orgID %s", orgID)
				return err
			}
			t.logger.Errorf("failed to find tasks in task_queue, orgID %s, error: %v", orgID, err)
			return err
		}
		eventIDs := make([]string, 0, len(events))
		for _, event := range events {
			jobs = append(jobs, core.FromQueueableEventToJob(event))
			eventIDs = append(eventIDs, event.ID)
		}
		count, err := t.eventQueueStore.UpdateBulkByIDInTx(ctx, tx, core.Processing, eventIDs...)
		if err != nil {
			t.logger.Errorf("failed update status of tasks in database, orgID %s, error: %v",
				orgID, err)
			return err
		}

		key := utils.GetOrgHashKey(orgID)
		t.logger.Debugf("Incrementing running_tasks by %d and queued_tasks by -%[1]d for key: %s, orgID %s",
			count, key, orgID)
		err = t.orgStore.UpdateKeysInCache(ctx, orgID, map[string]int64{"running_tasks": count, "queued_tasks": -count})
		if err != nil {
			t.logger.Errorf("failed to update conncurrency details for orgID %s, error: %v", orgID, err)
			return err
		}
		return nil
	})
}

func (t *taskQueueStore) UpdateTask(ctx context.Context, task *core.Task, orgID string) error {
	return t.db.ExecuteTransactionWithRetry(ctx, maxRetries, delay, maxJitter, errMsg, func(tx *sqlx.Tx) error {
		queueStatus := utils.QueueStatus(task.Status)
		alreadyFinished, err := t.eventQueueStore.FindAndUpdateInTx(ctx, tx, task.ID, core.TaskEvent, core.RefEntityTaskType, queueStatus)
		if err != nil {
			t.logger.Errorf("failed update event status in database, taskID %s, buildID %s, orgID %s, error: %v",
				task.ID, task.BuildID, orgID, err)
			return err
		}
		if err := t.taskStore.UpdateInTx(ctx, tx, task); err != nil {
			t.logger.Errorf("failed to update  task in datastore for taskID %s, buildID %s, orgID %s, error: %v",
				task.ID, task.BuildID, orgID, err)
			return err
		}
		// decrement count if task was not already finished in db, there by avoiding the count falling below 0
		if utils.TaskFinished(task.Status) && !alreadyFinished {
			now := time.Now()
			usage := &core.CreditsUsage{
				ID:      utils.GenerateUUID(),
				TaskID:  task.ID,
				OrgID:   orgID,
				Created: now,
				Updated: now,
			}
			if err := t.creditUsageStore.CreateInTx(ctx, tx, usage); err != nil {
				t.logger.Errorf("error while updating credit usage, taskID %s, buildID %s, orgID %s, error: %v",
					task.ID, task.BuildID, orgID, err)
				return err
			}
			t.logger.Debugf("Decrementing running_tasks by 1 orgID %s", orgID)
			if err := t.orgStore.UpdateKeysInCache(ctx, orgID, map[string]int64{"running_tasks": -1}); err != nil {
				return err
			}
			// Not tracking task count for discovery
			if task.Type == core.DiscoveryTask {
				return nil
			}
			t.logger.Debugf("Updating keys for taskID %s, buildID %s, orgID %s with status %s",
				task.ID, task.BuildID, orgID, task.Status)
			buildKey := utils.GetBuildHashKey(task.BuildID)
			if task.Type == core.FlakyTask {
				// update flaky task exec counts
				if err := t.updateTaskFlakyCount(ctx, buildKey, task); err != nil {
					t.logger.Errorf("error while updating keys in redis for taskID %s, buildID %s, orgID %s, error %v",
						task.ID, task.BuildID, orgID, err)
					return err
				}
				return nil
			}

			if err := t.updateTaskExecCount(ctx, buildKey, task); err != nil {
				t.logger.Errorf("error while updating keys in redis for taskID %s, buildID %s, orgID %s, error %v",
					task.ID, task.BuildID, orgID, err)
				return err
			}
		}

		return nil
	})
}

func (t *taskQueueStore) MarkError(ctx context.Context, buildID, orgID, remark string, jobs []*core.Job) error {
	return t.db.ExecuteTransactionWithRetry(ctx, maxRetries, delay, maxJitter, errMsg, func(tx *sqlx.Tx) error {
		eventIDs := make([]string, 0, len(jobs))
		for _, job := range jobs {
			eventIDs = append(eventIDs, job.ID)
		}
		count, err := t.eventQueueStore.UpdateBulkByIDInTx(ctx, tx, core.Failed, eventIDs...)
		if err != nil {
			t.logger.Errorf("failed to update mark tasks as failed in task queue, orgID %s, build %s, error: %v",
				orgID, buildID, err)
			return err
		}
		build := &core.Build{
			ID:      buildID,
			Updated: time.Now(),
			Status:  core.BuildError,
			Remark:  zero.StringFrom(remark),
		}
		// mark all tasks as error
		t.logger.Debugf("marking all tasks for buildID %s, orgID %s with status as %s", build.ID, orgID, build.Status)
		if err := t.taskStore.UpdateByBuildIDInTx(ctx, tx, core.TaskError, remark, buildID); err != nil {
			t.logger.Errorf("failed to mark all tasks as error for orgID %s, buildID %s, error %v",
				orgID, buildID, err)
			return err
		}
		// mark build as error
		t.logger.Debugf("marking buildID %s as stopped with status %s", build.ID, build.Status)
		if err := t.buildStore.MarkStoppedInTx(ctx, tx, build, orgID); err != nil {
			t.logger.Errorf("failed to mark build as error %s, error %v", buildID, err)
			return err
		}
		if count > 0 {
			orgKey := utils.GetOrgHashKey(orgID)
			t.logger.Debugf("Decrementing queued_tasks for key: %s by %d, orgID %s, buildID %s,", orgKey, count, orgID, buildID)
			if err := t.redisDB.Client().HIncrBy(context.Background(), orgKey, "queued_tasks", -1*count).Err(); err != nil {
				t.logger.Errorf("failed to update conncurrency details in redis for orgID %s, buildID %s, error: %v", orgID, buildID, err)
				return err
			}
		}
		return nil
	})
}

func (t *taskQueueStore) updateTaskExecCount(ctx context.Context, buildKey string, task *core.Task) error {
	if _, err := t.redisDB.Client().Pipelined(context.Background(), func(pipe redis.Pipeliner) error {
		// nolint:exhaustive
		switch task.Status {
		case core.TaskAborted:
			pipe.HIncrBy(ctx, buildKey, core.ExecTasksAborted, 1)
		case core.TaskFailed:
			pipe.HIncrBy(ctx, buildKey, core.ExecTasksFailed, 1)
		case core.TaskError:
			pipe.HIncrBy(ctx, buildKey, core.ExecTasksError, 1)
		case core.TaskPassed:
			pipe.HIncrBy(ctx, buildKey, core.ExecTasksPassed, 1)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (t *taskQueueStore) updateTaskFlakyCount(ctx context.Context, buildKey string, task *core.Task) error {
	if _, err := t.redisDB.Client().Pipelined(context.Background(), func(pipe redis.Pipeliner) error {
		// nolint:exhaustive
		switch task.Status {
		case core.TaskAborted:
			pipe.HIncrBy(ctx, buildKey, core.FlakyTasksAborted, 1)
		case core.TaskFailed:
			pipe.HIncrBy(ctx, buildKey, core.FlakyTasksFailed, 1)
		case core.TaskError:
			pipe.HIncrBy(ctx, buildKey, core.FlakyTasksError, 1)
		case core.TaskPassed:
			pipe.HIncrBy(ctx, buildKey, core.FlakyTasksPassed, 1)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}
