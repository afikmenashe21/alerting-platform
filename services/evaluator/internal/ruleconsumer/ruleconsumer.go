// Package ruleconsumer provides Kafka consumer functionality for rule.changed topic.
package ruleconsumer

import (
	"context"
	"fmt"
	"log/slog"

	kafkautil "github.com/afikmenashe/alerting-platform/pkg/kafka"
	protocommon "github.com/afikmenashe/alerting-platform/pkg/proto/common"
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

// fromProtoRuleAction converts a protobuf RuleAction enum to the simple action string.
func fromProtoRuleAction(action protocommon.RuleAction) string {
	switch action {
	case protocommon.RuleAction_RULE_ACTION_CREATED:
		return "CREATED"
	case protocommon.RuleAction_RULE_ACTION_UPDATED:
		return "UPDATED"
	case protocommon.RuleAction_RULE_ACTION_DELETED:
		return "DELETED"
	case protocommon.RuleAction_RULE_ACTION_DISABLED:
		return "DISABLED"
	default:
		return ""
	}
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

	// Log config from centralized source
	kafkautil.LogReaderConfig()

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
		Action:        fromProtoRuleAction(pb.Action), // Convert protobuf enum to simple action string
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
