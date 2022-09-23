package postprocessingqueue

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/LambdaTest/neuron/config"
	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/segmentio/kafka-go"
)

type producer struct {
	topicName         string
	kafkaWriter       *kafka.Writer
	logger            lumber.Logger
	producerStartTime time.Time
}

// NewProducer return a new post processing queue producer.
func NewProducer(cfg *config.Config,
	logger lumber.Logger) core.QueueProducer {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:          strings.Split(cfg.Kafka.Brokers, ","),
		Topic:            cfg.Kafka.PostProcessingQueueConfig.Topic,
		ErrorLogger:      kafka.LoggerFunc(logger.Errorf),
		Balancer:         &kafka.RoundRobin{},
		CompressionCodec: kafka.Snappy.Codec(),
		RequiredAcks:     int(kafka.RequireOne), // will wait for acknowledgement from only master.
	})
	logger.Infof("Kafka Producer connection created successfully for topic %s", writer.Topic)
	return &producer{
		logger:            logger,
		topicName:         writer.Topic,
		producerStartTime: time.Now(),
		kafkaWriter:       writer,
	}
}

// for post processing we don't have control dequeing, currently controlled by kafka itself.
func (p *producer) Enqueue(item interface{}) error {
	payload, ok := item.(*core.PostProcessingQueuePayload)
	if !ok {
		p.logger.Errorf("Invalid post processing queue payload %v", item)
		return errs.ErrInvalidQueuePayload
	}
	rawMessage, err := json.Marshal(payload)
	if err != nil {
		p.logger.Errorf("failed to marshal message for orgID %s, buildID %s, error: %v", payload.OrgID, payload.BuildID, err)
		return err
	}
	msg := kafka.Message{Value: rawMessage}
	if err = p.kafkaWriter.WriteMessages(context.Background(), msg); err != nil {
		p.logger.Errorf("failed to write message in kafka topic %s, orgID %s, buildID %s, error: %v",
			p.topicName, payload.OrgID, payload.BuildID, err)
		return err
	}
	return nil
}

func (p *producer) Close() error {
	return p.kafkaWriter.Close()
}
