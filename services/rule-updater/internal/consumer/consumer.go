// Package consumer provides Kafka consumer functionality for rule.changed topic.
package consumer

import (
	"context"
	"fmt"
	"log/slog"

	kafkautil "github.com/afikmenashe/alerting-platform/pkg/kafka"
	protocommon "github.com/afikmenashe/alerting-platform/pkg/proto/common"
	protorules "github.com/afikmenashe/alerting-platform/pkg/proto/rules"
	"rule-updater/internal/events"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

// MessageConsumer defines the interface for consuming rule change messages.
// This interface is implemented by Consumer and can be used for testing.
type MessageConsumer interface {
	ReadMessage(ctx context.Context) (*events.RuleChanged, *kafka.Message, error)
	CommitMessage(ctx context.Context, msg *kafka.Message) error
	Close() error
}

// Consumer wraps a Kafka reader and provides a simple interface for consuming rule.changed events.
type Consumer struct {
	reader *kafka.Reader
	topic  string
}

// fromProtoRuleAction converts a protobuf RuleAction enum to the typed Action.
func fromProtoRuleAction(action protocommon.RuleAction) events.Action {
	switch action {
	case protocommon.RuleAction_RULE_ACTION_CREATED:
		return events.ActionCreated
	case protocommon.RuleAction_RULE_ACTION_UPDATED:
		return events.ActionUpdated
	case protocommon.RuleAction_RULE_ACTION_DELETED:
		return events.ActionDeleted
	case protocommon.RuleAction_RULE_ACTION_DISABLED:
		return events.ActionDisabled
	default:
		return events.Action("")
	}
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

	// Log config from centralized source
	kafkautil.LogReaderConfig()

	return &Consumer{
		reader: reader,
		topic:  topic,
	}, nil
}

// ReadMessage reads the next message from Kafka and deserializes it as a RuleChanged.
// Returns an error if reading or deserialization fails.
func (c *Consumer) ReadMessage(ctx context.Context) (*events.RuleChanged, *kafka.Message, error) {
	msg, err := c.reader.ReadMessage(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read message from Kafka: %w", err)
	}

	var pb protorules.RuleChanged
	if err := proto.Unmarshal(msg.Value, &pb); err != nil {
		return nil, &msg, fmt.Errorf("failed to unmarshal protobuf rule.changed event: %w", err)
	}

	ruleChanged := events.RuleChanged{
		RuleID:        pb.RuleId,
		ClientID:      pb.ClientId,
		Action:        fromProtoRuleAction(pb.Action), // Convert protobuf enum to simple action string
		Version:       int(pb.Version),
		UpdatedAt:     pb.UpdatedAt,
		SchemaVersion: int(pb.SchemaVersion),
	}

	return &ruleChanged, &msg, nil
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
