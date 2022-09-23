package taskqueue

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	errs "github.com/LambdaTest/neuron/pkg/errors"

	"github.com/LambdaTest/neuron/config"
	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/segmentio/kafka-go"
)

type manager struct {
	kafkaWriter       *kafka.Writer
	logger            lumber.Logger
	producerStartTime time.Time
	taskQueueStore    core.TaskQueueStore
	orgStore          core.OrganizationStore
	buildStore        core.BuildStore
	taskStore         core.TaskStore
	gitStatusService  core.GitStatusService
}

// NewManager return a new task queue manager.
func NewManager(cfg *config.Config,
	taskQueueStore core.TaskQueueStore,
	orgStore core.OrganizationStore,
	buildStore core.BuildStore,
	taskStore core.TaskStore,
	gitStatusService core.GitStatusService,
	logger lumber.Logger) core.TaskQueueManager {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:          strings.Split(cfg.Kafka.Brokers, ","),
		Topic:            cfg.Kafka.TaskQueueConfig.Topic,
		ErrorLogger:      kafka.LoggerFunc(logger.Errorf),
		Balancer:         &kafka.RoundRobin{},
		CompressionCodec: kafka.Snappy.Codec(),
		RequiredAcks:     int(kafka.RequireOne), // will wait for acknowledgement from only master.
	})
	logger.Infof("Kafka Producer connection created successfully for topic %s", writer.Topic)
	return &manager{
		logger:            logger,
		producerStartTime: time.Now(),
		kafkaWriter:       writer,
		buildStore:        buildStore,
		taskStore:         taskStore,
		gitStatusService:  gitStatusService,
		taskQueueStore:    taskQueueStore,
		orgStore:          orgStore,
	}
}

func (m *manager) EnqueueTasks(orgID, buildID string, jobs ...*core.Job) error {
	if err := m.taskQueueStore.Create(context.Background(), orgID, buildID, jobs...); err != nil {
		m.logger.Errorf("error while inserting jobs in database orgID %s, buildID %s, error: %v",
			orgID, buildID, err)
		return err
	}
	go func() {
		if err := m.DequeueTasks(orgID); err != nil {
			m.logger.Errorf("error while dequeueing tasks from queue for orgID %s, buildID %s, error: %v", buildID, orgID, err)
		}
	}()
	return nil
}

func (m *manager) DequeueTasks(orgID string) error {
	ctx := context.Background()
	orgCache, err := m.orgStore.GetOrgCache(ctx, orgID)
	if err != nil {
		m.logger.Errorf("error while scanning redis hash orgID %s, error: %v", orgID, err)
		return err
	}
	if orgCache.RunningTasks >= orgCache.TotalConcurrency {
		m.logger.Infof("concurrency reached for orgID %s, current status %+v", orgID, orgCache)
		return nil
	}
	count := orgCache.TotalConcurrency - orgCache.RunningTasks
	// should not happen
	if count < 0 {
		m.logger.Errorf("total task to run is less than 0 !!!, orgID %s", orgID)
		count = orgCache.TotalConcurrency
	}

	jobs, err := m.taskQueueStore.FindAndUpdateTasks(ctx, orgID, count)
	if err != nil {
		if errors.Is(err, errs.ErrRowsNotFound) {
			m.logger.Infof("no tasks found in task queue for orgID %s", orgID)
			return nil
		}
		m.logger.Errorf("failed to dequeue tasks from database orgID %s, error: %v", orgID, err)
		return err
	}
	messages := make([]kafka.Message, 0, len(jobs))
	for _, job := range jobs {
		m.logger.Debugf("queuing job %s, for orgID %s, taskID %s", job.ID, orgID, job.TaskID)
		rawMessage, merr := json.Marshal(job)
		if merr != nil {
			m.logger.Errorf("failed to marshal message for orgID %s, job %s, taskID %s, error: %v",
				orgID, job.ID, job.TaskID, merr)
			return err
		}
		messages = append(messages, kafka.Message{Value: rawMessage})
	}
	if err = m.kafkaWriter.WriteMessages(ctx, messages...); err != nil {
		m.logger.Errorf("failed to write message in kafka, topic %s, orgID %s, error: %v", m.kafkaWriter.Topic, orgID, err)
		m.handleErrorJobs(ctx, orgID, jobs)
		return err
	}
	m.logger.Debugf("queued %d tasks for orgID %s", len(messages), orgID)
	return nil
}

func (m *manager) Close() error {
	return m.kafkaWriter.Close()
}

func (m *manager) handleErrorJobs(ctx context.Context, orgID string, jobs []*core.Job) {
	buildJobMap := map[string][]*core.Job{}
	for _, job := range jobs {
		task, err := m.taskStore.Find(ctx, job.TaskID)
		if err != nil {
			m.logger.Errorf("error in finding task for taskID %s, %v", job.TaskID, err)
			return
		}
		buildJobMap[task.BuildID] = append(buildJobMap[task.BuildID], job)
	}
	for buildID, jobs := range buildJobMap {
		buildCache, cerr := m.buildStore.GetBuildCache(ctx, buildID)
		if cerr != nil {
			m.logger.Errorf("error in finding build cache for buildID %s, %v", buildID, cerr)
			return
		}
		// mark task and build as failed if error while dequeueing a job.
		m.markBuildAndTasksAsError(buildID, orgID, jobs, buildCache)
	}
}

func (m *manager) markBuildAndTasksAsError(buildID,
	orgID string,
	jobs []*core.Job,
	buildCache *core.BuildCache) {
	remark := errs.GenericUserFacingBEErrRemark.Error()
	err := m.taskQueueStore.MarkError(context.Background(), buildID, orgID, remark, jobs)
	if err != nil {
		m.logger.Errorf("error while marking tasks and build as error for orgID %s, buildID %s, error: %v",
			orgID, buildID, err)
		return
	}
	if buildCache != nil {
		go func() {
			if errC := m.gitStatusService.UpdateGitStatus(context.Background(),
				buildCache.GitProvider,
				buildCache.RepoSlug,
				buildID,
				buildCache.TargetCommitID,
				buildCache.TokenPath,
				buildCache.InstallationTokenPath,
				core.BuildError,
				buildCache.BuildTag); errC != nil {
				m.logger.Errorf("error while updating status on GitHub for commitID: %s, buildID: %s, orgID: %s, err: %v",
					buildCache.TargetCommitID, buildID, orgID, errC)
			}
		}()
	}
}
