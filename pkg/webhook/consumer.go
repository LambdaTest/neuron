package webhook

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/LambdaTest/neuron/config"
	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/segmentio/kafka-go"
)

const (
	// EventHeader is the kafka header for webhook event type.
	EventHeader = "event_type"
	// RepoIDHeader is the kafka header for webhook repo_id.
	RepoIDHeader = "repo_id"
	// GitSCMHeader is the kafka header for webhook git_scm provider.
	GitSCMHeader = "git_driver"
)

type webhook struct {
	topicName string
	reader    *kafka.Reader
	logger    lumber.Logger
	startTime time.Time
	parser    core.HookParser
}

// New return a new kafka producer.
func New(cfg *config.Config, parser core.HookParser, logger lumber.Logger) core.QueueConsumer {
	// configure group balancer to RR
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:               strings.Split(cfg.Kafka.Brokers, ","),
		Topic:                 cfg.Kafka.WebhookConfig.Topic,
		ErrorLogger:           kafka.LoggerFunc(logger.Errorf),
		GroupID:               cfg.Kafka.WebhookConfig.ConsumerGroup,
		MaxBytes:              25e6, // 25MB
		WatchPartitionChanges: true,
		GroupBalancers:        []kafka.GroupBalancer{kafka.RoundRobinGroupBalancer{}}})
	// offset retention time is 24h
	logger.Infof("Kafka Consumer Group %s created successfully", cfg.Kafka.WebhookConfig.ConsumerGroup)
	return &webhook{
		topicName: cfg.Kafka.WebhookConfig.Topic,
		logger:    logger,
		startTime: time.Now(),
		reader:    reader,
		parser:    parser,
	}
}

func (w *webhook) Run(ctx context.Context) {
	for {
		msg, err := w.reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				break
			}
			w.logger.Errorf("Kafka ReadMessage of topic: %v failed: %v", w.topicName, err)
			continue
		}
		w.logger.Debugf("Kafka: Message received on partition: %d, offset: %d, topic: %s", msg.Partition, msg.Offset, msg.Topic)
		// perform tasks in go routine
		go func(msg kafka.Message) {
			eventType, driver, repoID := w.getDataFromHeaders(msg.Headers)
			if eventType == "" || driver == "" || repoID == "" {
				w.logger.Errorf("Kafka: Invalid Headers received on topic: %s, partition: %d, offset: %d", msg.Topic, msg.Partition, msg.Offset)
				return
			}
			// parse hook payload
			err := w.parser.CreateAndScheduleBuild(eventType, driver, repoID, msg.Value)
			if err != nil {
				w.logger.Errorf("failed to parse message, repoID %s, error: %v", repoID, err)
			}
		}(msg)
	}

	if err := w.Close(); err != nil {
		w.logger.Errorf("failed to close Kafka reader, error: %v", err)
	}
}

func (w *webhook) Close() error {
	return w.reader.Close()
}

func (w *webhook) getDataFromHeaders(headers []kafka.Header) (eventType core.EventType, driver core.SCMDriver, repoID string) {
	for _, header := range headers {
		switch header.Key {
		case EventHeader:
			eventType = core.EventType(string(header.Value))
		case RepoIDHeader:
			repoID = string(header.Value)
		case GitSCMHeader:
			driver = core.SCMDriver(string(header.Value))
		}
	}
	return eventType, driver, repoID
}
