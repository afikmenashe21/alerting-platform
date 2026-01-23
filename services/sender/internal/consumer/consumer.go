// Package consumer provides Kafka consumer functionality for notifications.ready topic.
package consumer

import (
	"context"
	"fmt"
	"log/slog"

	kafkautil "github.com/afikmenashe/alerting-platform/pkg/kafka"
	pbnotifications "github.com/afikmenashe/alerting-platform/pkg/proto/notifications"
	"sender/internal/events"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

// Consumer wraps a Kafka reader and provides a simple interface for consuming notification ready events.
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
		"min_bytes", 1,
		"max_bytes", 10e6,
		"max_wait", kafkautil.MaxPollWait,
		"commit_interval", kafkautil.CommitInterval,
	)

	return &Consumer{
		reader: reader,
		topic:  topic,
	}, nil
}

// ReadMessage reads the next message from Kafka and deserializes it as a NotificationReady.
// Returns an error if reading or deserialization fails.
func (c *Consumer) ReadMessage(ctx context.Context) (*events.NotificationReady, *kafka.Message, error) {
	msg, err := c.reader.ReadMessage(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read message from Kafka: %w", err)
	}

	var pb pbnotifications.NotificationReady
	if err := proto.Unmarshal(msg.Value, &pb); err != nil {
		return nil, &msg, fmt.Errorf("failed to unmarshal notification ready protobuf: %w", err)
	}

	ready := &events.NotificationReady{
		NotificationID: pb.NotificationId,
		ClientID:       pb.ClientId,
		AlertID:        pb.AlertId,
		SchemaVersion:  int(pb.SchemaVersion),
	}

	return ready, &msg, nil
}

// CommitMessage commits the offset for the given message.
// This should be called after successfully processing a message.
func (c *Consumer) CommitMessage(ctx context.Context, msg *kafka.Message) error {
	return c.reader.CommitMessages(ctx, *msg)
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
