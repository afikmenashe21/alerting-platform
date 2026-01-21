// Package consumer provides Kafka consumer functionality for alerts.new topic.
package consumer

import (
	"context"
	"fmt"
	"log/slog"

	kafkautil "github.com/afikmenashe/alerting-platform/pkg/kafka"
	pbalerts "github.com/afikmenashe/alerting-platform/pkg/proto/alerts"
	"evaluator/internal/events"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

// Consumer wraps a Kafka reader and provides a simple interface for consuming alerts.
type Consumer struct {
	reader *kafka.Reader
	topic  string
}

// NewConsumer creates a new Kafka consumer with the specified brokers, topic, and group ID.
// The consumer is configured for at-least-once delivery semantics.
func NewConsumer(brokers string, topic string, groupID string) (*Consumer, error) {
	if err := kafkautil.ValidateConsumerParams(brokers, topic, groupID); err != nil {
		return nil, err
	}

	// Parse comma-separated broker list
	brokerList := kafkautil.ParseBrokers(brokers)

	slog.Info("Initializing Kafka consumer",
		"brokers", brokerList,
		"topic", topic,
		"group_id", groupID,
	)

	// Configure Kafka reader for at-least-once delivery
	// StartOffset only applies when no committed offset exists for the consumer group
	// Using FirstOffset ensures we read all messages when starting fresh
	reader := kafka.NewReader(kafkautil.NewReaderConfig(brokerList, topic, groupID))

	slog.Info("Kafka consumer configured",
		"min_bytes", 10e3,
		"max_bytes", 10e6,
		"max_wait", kafkautil.ReadTimeout,
		"commit_interval", kafkautil.CommitInterval,
	)

	return &Consumer{
		reader: reader,
		topic:  topic,
	}, nil
}

// ReadMessage reads the next message from Kafka and deserializes it as an AlertNew.
// Returns an error if reading or deserialization fails.
func (c *Consumer) ReadMessage(ctx context.Context) (*events.AlertNew, *kafka.Message, error) {
	msg, err := c.reader.ReadMessage(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read message from Kafka: %w", err)
	}

	var pb pbalerts.AlertNew
	if err := proto.Unmarshal(msg.Value, &pb); err != nil {
		return nil, &msg, fmt.Errorf("failed to unmarshal alert protobuf: %w", err)
	}

	alert := &events.AlertNew{
		AlertID:       pb.AlertId,
		SchemaVersion: int(pb.SchemaVersion),
		EventTS:       pb.EventTs,
		Severity:      pb.Severity.String(),
		Source:        pb.Source,
		Name:          pb.Name,
		Context:       pb.Context,
	}

	return alert, &msg, nil
}

// Close gracefully closes the Kafka reader and releases resources.
func (c *Consumer) Close() error {
	slog.Info("Closing Kafka consumer", "topic", c.topic)
	if err := c.reader.Close(); err != nil {
		slog.Error("Error closing Kafka consumer", "error", err)
		return err
	}
	slog.Info("Kafka consumer closed successfully")
	return nil
}
