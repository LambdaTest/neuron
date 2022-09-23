package synapsequeue

import (
	"context"
	"encoding/json"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/synapse"
	"github.com/LambdaTest/neuron/pkg/ws"
)

type synapseQueue struct {
	logger         lumber.Logger
	redisDB        core.RedisDB
	channelName    string
	synapseManager core.SynapseClientManager
	synapsePool    core.SynapsePoolManager
	synapseStore   core.SynapseStore
}

// New returns new synapseQueue
func New(redisDB core.RedisDB,
	logger lumber.Logger,
	synapseManager core.SynapseClientManager,
	synapsePool core.SynapsePoolManager,
	synapseStore core.SynapseStore) core.SynapseQueueManager {
	return &synapseQueue{redisDB: redisDB,
		logger:         logger,
		channelName:    "synapseTaskQueue",
		synapseManager: synapseManager,
		synapsePool:    synapsePool,
		synapseStore:   synapseStore,
	}
}

func (s *synapseQueue) publish(ctx context.Context, synapseID string, runnerOpts *core.RunnerOptions) error {
	synapseTask := core.SynapseTask{
		SynapseID: synapseID,
		Task:      runnerOpts,
	}
	synapseTaskJSON, err := json.Marshal(synapseTask)
	if err != nil {
		return err
	}
	err = s.redisDB.Client().Publish(ctx, s.channelName, synapseTaskJSON).Err()
	if err != nil {
		return err
	}
	return nil
}

// Run subscribes to the synapse task topics
func (s *synapseQueue) Run(ctx context.Context) {
	subscriber := s.redisDB.Client().Subscribe(ctx, s.channelName)
	_, err := subscriber.Receive(ctx)
	if err != nil {
		return
	}
	go func() {
		<-ctx.Done()
		subscriber.Close()
		s.logger.Debugf("exiting synapse subscriber")
	}()
	subscriptionChan := subscriber.Channel()
	for msg := range subscriptionChan {
		var synapseTask core.SynapseTask
		if err := json.Unmarshal([]byte(msg.Payload), &synapseTask); err != nil {
			continue
		}
		synapseClient, exists := s.synapsePool.GetSynapseFromID(synapseTask.SynapseID)
		if exists {
			taskMessage := ws.CreateTaskMessage(synapseTask.Task)
			if err := s.synapseManager.SendMessage(synapseClient, &taskMessage); err != nil {
				s.logger.Errorf("error sending message to synapseID %s orgID %s error:%v", synapseClient.ID, synapseClient.OrgID, err)

				spec := synapse.GetResources(synapseTask.Task.Tier)
				if err := s.synapseManager.ReleaseResources(synapseClient, spec.CPU, spec.RAM); err != nil {
					s.logger.Errorf("error occurred while releasing the resource for synapseID %s orgID %s  error %v ",
						synapseClient.ID, synapseClient.OrgID, err)
				}

				s.updateJobStatus(&synapseTask, synapseClient, core.JobFailed)
				continue
			}

			s.updateJobStatus(&synapseTask, synapseClient, core.JobInitiated)
		} else {
			s.logger.Errorf("synapse not found for orgID %s buildID %s jobID %s ", synapseTask.Task.OrgID,
				synapseTask.Task.Label[synapse.BuildID], synapseTask.Task.Label[synapse.JobID])
		}
	}
}

// ScheduleTask finds executor and publishes message
func (s *synapseQueue) ScheduleTask(r *core.RunnerOptions, synapseID string) error {
	if err := s.publish(context.TODO(), synapseID, r); err != nil {
		s.logger.Errorf("error publishing task for synapse", err)
		return err
	}
	return nil
}

func (s *synapseQueue) updateJobStatus(synapseTask *core.SynapseTask, synapseClient *core.SynapseMeta, status core.StatusType) {
	label := synapseTask.Task.Label
	jobInfo := &core.JobInfo{
		Status:  status,
		JobID:   label[synapse.JobID],
		ID:      label[synapse.ID],
		Mode:    label[synapse.Mode],
		BuildID: label[synapse.BuildID],
	}
	if err := s.synapseManager.UpdateJobStatus(synapseClient, jobInfo); err != nil {
		s.logger.Errorf("failed to update job status to redis for synapse %s orgID %s buildID %s jobID %s ",
			synapseClient.ID, synapseClient.OrgID, synapse.BuildID, synapse.JobID)
	}
}
