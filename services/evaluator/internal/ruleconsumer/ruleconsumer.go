// Package ruleconsumer provides Kafka consumer functionality for rule.changed topic.
package ruleconsumer

import (
	"context"
	"fmt"
	"log/slog"

	kafkautil "github.com/afikmenashe/alerting-platform/pkg/kafka"
	protorules "github.com/afikmenashe/alerting-platform/pkg/proto/rules"
	"evaluator/internal/events"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

// Consumer wraps a Kafka reader for consuming rule.changed events.
type Consumer struct {
	reader *kafka.Reader
	topic  string
}

// NewConsumer creates a new Kafka consumer for rule.changed topic.
func NewConsumer(brokers string, topic string, groupID string) (*Consumer, error) {
	if err := kafkautil.ValidateConsumerParams(brokers, topic, groupID); err != nil {
		return nil, err
	}

	// Parse comma-separated broker list
	brokerList := kafkautil.ParseBrokers(brokers)

	slog.Info("Initializing rule.changed Kafka consumer",
		"brokers", brokerList,
		"topic", topic,
		"group_id", groupID,
	)

	// Configure Kafka reader for at-least-once delivery
	reader := kafka.NewReader(kafkautil.NewReaderConfig(brokerList, topic, groupID))

	slog.Info("Rule.changed consumer configured",
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

// ReadMessage reads the next rule.changed message from Kafka.
func (c *Consumer) ReadMessage(ctx context.Context) (*events.RuleChanged, error) {
	msg, err := c.reader.ReadMessage(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read message from Kafka: %w", err)
	}

	var pb protorules.RuleChanged
	if err := proto.Unmarshal(msg.Value, &pb); err != nil {
		return nil, fmt.Errorf("failed to unmarshal protobuf rule changed event: %w", err)
	}

	return &events.RuleChanged{
		RuleID:        pb.RuleId,
		ClientID:      pb.ClientId,
		Action:        pb.Action.String(), // existing code expects string
		Version:       int(pb.Version),
		UpdatedAt:     pb.UpdatedAt,
		SchemaVersion: int(pb.SchemaVersion),
	}, nil
}

// Close gracefully closes the Kafka reader.
func (c *Consumer) Close() error {
	slog.Info("Closing rule.changed consumer", "topic", c.topic)
	if err := c.reader.Close(); err != nil {
		slog.Error("Error closing rule.changed consumer", "error", err)
		return err
	}
	slog.Info("Rule.changed consumer closed successfully")
	return nil
}
