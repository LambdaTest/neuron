package buildabortqueue

import (
	"context"
	"strings"

	"github.com/LambdaTest/neuron/config"
	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/segmentio/kafka-go"
)

type producer struct {
	kafkaWriter *kafka.Writer
	topicName   string
	logger      lumber.Logger
}

// NewProducer returns producer for the build-abort
func NewProducer(cfg *config.Config,
	logger lumber.Logger) core.BuildAbortProducer {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:          strings.Split(cfg.Kafka.Brokers, ","),
		Topic:            cfg.Kafka.BuildAbortConfig.Topic,
		ErrorLogger:      kafka.LoggerFunc(logger.Errorf),
		Balancer:         &kafka.RoundRobin{},
		CompressionCodec: kafka.Snappy.Codec(),
		RequiredAcks:     int(kafka.RequireOne), // will wait for acknowledgement from only master.
	})
	logger.Debugf("Kafka Producer connection created successfully for topic %s", writer.Topic)
	return &producer{
		logger:      logger,
		kafkaWriter: writer,
		topicName:   writer.Topic,
	}
}

func (p *producer) Enqueue(buildID, orgID string) error {
	msg := kafka.Message{Value: []byte(buildID)}
	if err := p.kafkaWriter.WriteMessages(context.Background(), msg); err != nil {
		p.logger.Errorf("failed to write message on kafka topic: %s for buildID: %s, and orgID: %s, error: %v", p.topicName, buildID, orgID, err)
		return err
	}
	return nil
}

func (p *producer) Close() error {
	return p.kafkaWriter.Close()
}
