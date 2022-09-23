package buildabortqueue

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/LambdaTest/neuron/config"
	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/segmentio/kafka-go"
	"golang.org/x/sync/errgroup"
)

type consumer struct {
	cfg               *config.Config
	topicName         string
	reader            *kafka.Reader
	logger            lumber.Logger
	buildStore        core.BuildStore
	taskStore         core.TaskStore
	taskUpdateManager core.TaskUpdateManager
	spawnedPodsMap    core.SyncMap
}

// NewConsumer return a new BuildAbort consumer.
func NewConsumer(cfg *config.Config,
	buildStore core.BuildStore,
	taskStore core.TaskStore,
	taskUpdateManager core.TaskUpdateManager,
	spawnedPodsMap core.SyncMap,
	logger lumber.Logger) core.BuildAbortConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:               strings.Split(cfg.Kafka.Brokers, ","),
		Topic:                 cfg.Kafka.BuildAbortConfig.Topic,
		ErrorLogger:           kafka.LoggerFunc(logger.Errorf),
		GroupID:               fmt.Sprintf("%v-%v", cfg.Kafka.BuildAbortConfig.ConsumerGroup, utils.GenerateUUID()),
		MaxBytes:              25e6, // 25MB
		WatchPartitionChanges: true,
		GroupBalancers:        []kafka.GroupBalancer{kafka.RoundRobinGroupBalancer{}}})
	// offset retention time is 24h
	logger.Infof("Kafka Consumer Group %s created successfully", cfg.Kafka.BuildAbortConfig.ConsumerGroup)

	return &consumer{
		cfg:               cfg,
		topicName:         reader.Config().Topic,
		logger:            logger,
		reader:            reader,
		buildStore:        buildStore,
		taskStore:         taskStore,
		taskUpdateManager: taskUpdateManager,
		spawnedPodsMap:    spawnedPodsMap,
	}
}

func (c *consumer) Run(ctx context.Context) {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				break
			}
			c.logger.Errorf("kafka message of topic; %s failed: %v", c.topicName, err)
			continue
		}
		c.logger.Debugf("Kafka: Message received on partition: %d, offset: %d, topic: %s", msg.Partition, msg.Offset, msg.Topic)
		// perform tasks in go routine
		go func(msg kafka.Message) {
			buildID := string(msg.Value)
			if errC := c.terminateAllRunningPods(ctx, buildID); errC != nil {
				c.logger.Errorf("failed to terminate running pods for buildID: %s, orgID: %s, error: %v", buildID, errC)
			}
		}(msg)
	}
	if err := c.Close(); err != nil {
		c.logger.Errorf("failed to close Kafka reader, error: %v", err)
	}
	c.logger.Debugf("kafka consumer closed for build-abort topic")
}

func (c *consumer) Close() error {
	return c.reader.Close()
}

func (c *consumer) terminateAllRunningPods(ctx context.Context, buildID string) error {
	// fetch all running tasks
	tasks, err := c.taskStore.FetchTaskHavingStatus(ctx, buildID, core.TaskRunning)
	if err != nil {
		c.logger.Errorf("failed to fetch running tasks for buildID: %s, error: %v", buildID, err)
		return err
	}

	c.logger.Debugf("fetched running tasks count: %+v", len(tasks))
	g, _ := errgroup.WithContext(ctx)
	for _, task := range tasks {
		func(task *core.Task) {
			g.Go(func() error {
				taskID := task.ID
				key := utils.GetSpawnedPodsMapKey(buildID, taskID)
				c.logger.Debugf("checking and deleting key from spawnedPodsMap: %+v", key)
				c.spawnedPodsMap.CleanUp(key)
				return nil
			})
		}(task)
	}
	if err = g.Wait(); err != nil {
		c.logger.Errorf("error while aborting tasks for buildID: %s, error: %v", buildID, err)
	}
	return nil
}
