package postprocessingqueue

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/LambdaTest/neuron/config"
	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/segmentio/kafka-go"
)

type consumer struct {
	cfg             *config.Config
	topicName       string
	reader          *kafka.Reader
	logger          lumber.Logger
	runner          core.K8sRunner
	coverageManager core.CoverageManager
}

// NewConsumer return a new taskqueue consumer.
func NewConsumer(cfg *config.Config,
	coverageManager core.CoverageManager,
	runner core.K8sRunner,
	logger lumber.Logger) core.QueueConsumer {
	// offset retention time is 24h
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:               strings.Split(cfg.Kafka.Brokers, ","),
		Topic:                 cfg.Kafka.PostProcessingQueueConfig.Topic,
		ErrorLogger:           kafka.LoggerFunc(logger.Errorf),
		GroupID:               cfg.Kafka.PostProcessingQueueConfig.ConsumerGroup,
		WatchPartitionChanges: true,
		GroupBalancers:        []kafka.GroupBalancer{kafka.RoundRobinGroupBalancer{}}})
	logger.Infof("Kafka Consumer Group %s created successfully", reader.Config().GroupID)

	return &consumer{
		cfg:             cfg,
		topicName:       reader.Config().Topic,
		logger:          logger,
		reader:          reader,
		runner:          runner,
		coverageManager: coverageManager,
	}
}

func (c *consumer) Run(ctx context.Context) {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				break
			}
			c.logger.Errorf("Kafka ReadMessage of topic: %v failed: %v", c.topicName, err)
			continue
		}
		c.logger.Debugf("Kafka: Message received on partition: %d, offset: %d, topic: %s", msg.Partition, msg.Offset, msg.Topic)
		// perform tasks in go routine
		go func(msg kafka.Message) {
			var data core.PostProcessingQueuePayload
			if err := json.Unmarshal(msg.Value, &data); err != nil {
				c.logger.Errorf("failed to unmarshal job payload, error: %v", err)
				return
			}
			canRun, err := c.coverageManager.CanRunCoverageImmediately(ctx, data.BaseCommitID, data.RepoID)
			if err != nil {
				c.logger.Errorf("error checking if base-commit's %s, buildID %s orgID %s coverage is recorded or not. error: %v",
					data.BaseCommitID, data.BuildID, data.OrgID, err)
				runnerOpts := &core.RunnerOptions{
					NameSpace:                 utils.GetRunnerNamespaceFromOrgID(data.OrgID),
					PersistentVolumeClaimName: utils.GetBuildHashKey(data.BuildID)}
				// delete pvc if base commit not found for build.
				if pvcErr := c.runner.Cleanup(ctx, runnerOpts); pvcErr != nil {
					c.logger.Errorf("failed to delete pvc in k8s for buildID %s, orgID %s, error %v",
						data.BuildID, data.OrgID, pvcErr)
				}
				return
			}
			if !canRun {
				c.logger.Infof("cannot immediately run coverage job for buildID %s, orgID %s", data.BuildID, data.OrgID)
				return
			}
			coverageInput := &core.CoverageInput{BuildID: data.BuildID, OrgID: data.OrgID, PayloadAddress: data.PayloadAddress}
			if err := c.coverageManager.RunCoverageJob(ctx, coverageInput); err != nil {
				c.logger.Errorf("Error while running coverage job for buildID %s, orgID %s, error %v", data.BuildID, data.OrgID, err)
			}
		}(msg)
	}

	if err := c.Close(); err != nil {
		c.logger.Errorf("failed to closed kafka reader, error: %v", err)
		return
	}
	c.logger.Debugf("Kafka consumer closed successfully for topic %s", c.topicName)
}

func (c *consumer) Close() error {
	return c.reader.Close()
}
