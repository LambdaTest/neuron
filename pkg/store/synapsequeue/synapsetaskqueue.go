package synapsequeue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/synapse"
	"github.com/alphayan/redisqueue/v3"
)

type synapseTaskQueue struct {
	redisDB        core.RedisDB
	producer       *redisqueue.Producer
	consumer       *redisqueue.Consumer
	logger         lumber.Logger
	synapsePool    core.SynapsePoolManager
	synapseManager core.SynapseClientManager
	synapseQueue   core.SynapseQueueManager
	buildStore     core.BuildStore
}

const (
	visibilityTimeout    = 30
	blockingTimeout      = 5
	reclaimInterval      = 1
	streamMaxLength      = 1000
	concurrency          = 1
	bufferSize           = 1
	streamName           = "synapse:taskqueue"
	messageWaitHours     = 2.5
	noSynapseFoundRemark = "No synapse found for job, please make sure synapse is connected"
)

// NewSynapseTaskQueue creates queue producer and returns new synapseTaskQueue
func NewSynapseTaskQueue(redisDB core.RedisDB,
	synapsePool core.SynapsePoolManager,
	synapseManager core.SynapseClientManager,
	synapseQueue core.SynapseQueueManager,
	buildStore core.BuildStore,
	logger lumber.Logger) (core.SynapseTaskQueue, error) {
	producer, err := redisqueue.NewProducerWithOptions(
		&redisqueue.ProducerOptions{
			StreamMaxLength:      streamMaxLength,
			ApproximateMaxLength: true,
			RedisClient:          redisDB.Client(),
		},
	)
	if err != nil {
		return nil, err
	}

	consumer, err := redisqueue.NewConsumerWithOptions(
		&redisqueue.ConsumerOptions{
			VisibilityTimeout: visibilityTimeout * time.Second,
			BlockingTimeout:   blockingTimeout * time.Second,
			ReclaimInterval:   reclaimInterval * time.Second,
			Concurrency:       concurrency,
			RedisClient:       redisDB.Client(),
			BufferSize:        bufferSize,
		},
	)
	if err != nil {
		return nil, err
	}

	return &synapseTaskQueue{
		redisDB:        redisDB,
		producer:       producer,
		consumer:       consumer,
		logger:         logger,
		synapsePool:    synapsePool,
		synapseManager: synapseManager,
		synapseQueue:   synapseQueue,
		buildStore:     buildStore,
	}, nil
}

func (s *synapseTaskQueue) InitConsumer(ctx context.Context) {
	s.consumer.Register(streamName, s.scheduleOnSynapse)
	go func() {
		for {
			<-s.consumer.Errors
		}
	}()
	go func() {
		<-ctx.Done()
		s.consumer.Shutdown()
		s.logger.Debugf("closed synapse queue consumer")
	}()
	s.consumer.Run()
}

func (s *synapseTaskQueue) scheduleOnSynapse(msg *redisqueue.Message) error {
	taskJSON := msg.Values["runnerOptions"].(string)
	createdJSON := msg.Values["createdAt"].(string)
	var createdAt time.Time
	var task core.RunnerOptions
	if err := json.Unmarshal([]byte((taskJSON)), &task); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte((createdJSON)), &createdAt); err != nil {
		return err
	}
	synapseMeta, err := s.synapsePool.FindExecutor(&task)
	spec := synapse.GetResources(task.Tier)

	if err == nil {
		synapseMeta.OrgID = task.OrgID
		if err = s.synapseManager.CaptureResources(synapseMeta, spec.CPU, spec.RAM); err != nil {
			s.logger.Errorf("error capturing resources for synapseID %s orgID %s error:%v",
				synapseMeta.ID, synapseMeta.OrgID, err)
		}
	}
	if err != nil {
		if time.Since(createdAt).Hours() > messageWaitHours {
			// discard the message after time out and marked the job failed
			s.logger.Errorf("Timeout exceed for orgID %s buildID %s jobID %s , discarding the message and marking it as failure", task.OrgID,
				task.Label[synapse.BuildID], task.Label[synapse.JobID])
			jobInfo := &core.JobInfo{
				Status:  core.JobFailed,
				JobID:   task.Label[synapse.JobID],
				ID:      task.Label[synapse.ID],
				Mode:    task.Label[synapse.Mode],
				BuildID: task.Label[synapse.BuildID],
			}
			s.synapseManager.MarkJobFailed(jobInfo, noSynapseFoundRemark)
			return nil
		}
		errMsg := fmt.Sprintf("Applying visblity time out for buildID %s jobID %s no synapse available for orgID %s",
			task.Label[synapse.BuildID], task.Label[synapse.ID], task.OrgID)
		return errors.New(errMsg)
	}
	s.logger.Infof("synapse %s found for orgID %s buildID %s jobID %s", synapseMeta.ID, synapseMeta.OrgID,
		task.Label[synapse.BuildID], task.Label[synapse.JobID])

	build, err := s.buildStore.FindByBuildID(context.Background(), task.Label[synapse.BuildID])
	// not retrurning error as it will lead to visbility timeout for message and message will not be discarded
	if err != nil {
		s.logger.Errorf("Error while reading build entry from DB for buildID %s orgID %s, taskID %s",
			task.Label[synapse.BuildID], task.OrgID, task.Label[synapse.ID])
		s.releaseResources(synapseMeta, spec)
		return nil
	}
	// do not schedule the task if build is already failed/aborted/errored
	if build.Status != core.BuildRunning && build.Status != core.BuildInitiating {
		s.logger.Errorf("discarding the orgID %s buildID %s taskID %s jobID %s , as build is not in running state",
			task.OrgID, task.Label[synapse.BuildID], task.Label[synapse.ID], task.Label[synapse.JobID])
		s.releaseResources(synapseMeta, spec)
		return nil
	}
	if err := s.synapseQueue.ScheduleTask(&task, synapseMeta.ID); err != nil {
		errMsg := fmt.Sprintf("errors in publishing message for synapse %s  orgID %s buildID %s jobID %s",
			synapseMeta.ID, synapseMeta.OrgID, task.Label[synapse.BuildID], task.Label[synapse.JobID])
		s.releaseResources(synapseMeta, spec)
		return errors.New(errMsg)
	}
	return nil
}

func (s *synapseTaskQueue) releaseResources(synapseMeta *core.SynapseMeta, spec synapse.Specs) {
	if err := s.synapseManager.ReleaseResources(synapseMeta, spec.CPU, spec.RAM); err != nil {
		s.logger.Errorf("error occurred while releasing the resource for synapseID %s orgID %s , error %v",
			synapseMeta.ID, synapseMeta.OrgID, err)
	}
}

func (s *synapseTaskQueue) Enqueue(r *core.RunnerOptions, createdAt time.Time) <-chan error {
	errChan := make(chan error)
	defer close(errChan)
	task, err := json.Marshal(r)
	if err != nil {
		errChan <- err
		return errChan
	}
	createdTime, err := json.Marshal(createdAt)
	if err != nil {
		errChan <- err
		return errChan
	}
	err = s.producer.Enqueue(&redisqueue.Message{
		Stream: streamName,
		Values: map[string]interface{}{
			"runnerOptions": task,
			"createdAt":     createdTime,
		},
	})
	if err != nil {
		errChan <- err
		return errChan
	}
	return errChan
}
