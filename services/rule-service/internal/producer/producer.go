// Package producer provides Kafka producer functionality for rule.changed topic.
package producer

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	protorules "github.com/afikmenashe/alerting-platform/pkg/proto/rules"
	protocommon "github.com/afikmenashe/alerting-platform/pkg/proto/common"
	"rule-service/internal/events"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

const (
	// writeTimeout is the maximum time to wait for a Kafka write operation.
	writeTimeout = 10 * time.Second
)

// Producer wraps a Kafka writer and provides a simple interface for publishing rule changed events.
type Producer struct {
	writer *kafka.Writer
	topic  string
}

// NewProducer creates a new Kafka producer with the specified brokers and topic.
// The producer is configured for at-least-once delivery semantics with synchronous writes.
func NewProducer(brokers string, topic string) (*Producer, error) {
	if brokers == "" {
		return nil, fmt.Errorf("brokers cannot be empty")
	}
	if topic == "" {
		return nil, fmt.Errorf("topic cannot be empty")
	}

	// Parse comma-separated broker list
	brokerList := strings.Split(brokers, ",")
	for i := range brokerList {
		brokerList[i] = strings.TrimSpace(brokerList[i])
	}

	slog.Info("Initializing Kafka producer",
		"brokers", brokerList,
		"topic", topic,
	)

	// Try to create topic if it doesn't exist (best effort, may fail silently)
	createTopicIfNotExists(brokerList[0], topic)

	// Configure Kafka writer for at-least-once delivery
	// Use Hash balancer to partition by rule_id
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokerList...),
		Topic:        topic,
		Balancer:     &kafka.Hash{}, // Key-based partitioning (hashes the message key)
		WriteTimeout: writeTimeout,
		RequiredAcks: kafka.RequireOne, // At-least-once semantics (waits for leader ack)
		Async:        false,            // Synchronous writes for reliability and error handling
	}

	slog.Info("Kafka producer configured",
		"write_timeout", writeTimeout,
		"required_acks", "RequireOne",
		"async", false,
		"balancer", "Hash (key-based partitioning)",
		"partition_key", "rule_id (hashed)",
	)

	return &Producer{
		writer: writer,
		topic:  topic,
	}, nil
}


func toProtoRuleAction(action string) protocommon.RuleAction {
	switch action {
	case events.ActionCreated:
		return protocommon.RuleAction_RULE_ACTION_CREATED
	case events.ActionUpdated:
		return protocommon.RuleAction_RULE_ACTION_UPDATED
	case events.ActionDeleted:
		return protocommon.RuleAction_RULE_ACTION_DELETED
	case events.ActionDisabled:
		return protocommon.RuleAction_RULE_ACTION_DISABLED
	default:
		return protocommon.RuleAction_RULE_ACTION_UNSPECIFIED
	}
}

// Publish serializes a rule changed event to protobuf and publishes it to Kafka.
// The message is keyed by rule_id for partition distribution.
// Returns an error if serialization or publishing fails.
func (p *Producer) Publish(ctx context.Context, changed *events.RuleChanged) error {
	evt := &protorules.RuleChanged{
		RuleId:        changed.RuleID,
		ClientId:      changed.ClientID,
		Action:        toProtoRuleAction(changed.Action),
		Version:       int32(changed.Version),
		UpdatedAt:     changed.UpdatedAt,
		SchemaVersion: int32(changed.SchemaVersion),
	}

	payload, err := proto.Marshal(evt)
	if err != nil {
		slog.Error("Failed to marshal rule changed event to protobuf",
			"rule_id", changed.RuleID,
			"client_id", changed.ClientID,
			"action", changed.Action,
			"error", err,
		)
		return fmt.Errorf("failed to marshal rule changed event: %w", err)
	}

	// Partition key: use rule_id
	partitionKey := []byte(changed.RuleID)

	// Create Kafka message with key, value, headers, and timestamp
	msg := kafka.Message{
		Key:   partitionKey,
		Value: payload,
		Headers: []kafka.Header{
			{
				Key:   "content-type",
				Value: []byte("application/x-protobuf"),
			},
			{
				Key:   "schema_version",
				Value: []byte(fmt.Sprintf("%d", changed.SchemaVersion)),
			},
			{
				Key:   "action",
				Value: []byte(changed.Action),
			},
			{
				Key:   "rule_id",
				Value: []byte(changed.RuleID),
			},
		},
		Time: time.Unix(changed.UpdatedAt, 0),
	}

	// Write to Kafka (synchronous, waits for ack)
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		slog.Error("Failed to write message to Kafka",
			"rule_id", changed.RuleID,
			"topic", p.topic,
			"error", err,
		)
		return fmt.Errorf("failed to write message to Kafka: %w", err)
	}

	slog.Info("Published rule changed event",
		"rule_id", changed.RuleID,
		"client_id", changed.ClientID,
		"action", changed.Action,
		"version", changed.Version,
	)

	return nil
}

// Close gracefully closes the Kafka writer and releases resources.
func (p *Producer) Close() error {
	slog.Info("Closing Kafka producer", "topic", p.topic)
	if err := p.writer.Close(); err != nil {
		slog.Error("Error closing Kafka producer", "error", err)
		return err
	}
	slog.Info("Kafka producer closed successfully")
	return nil
}
