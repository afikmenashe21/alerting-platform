// Package producer provides Kafka producer functionality for rule.changed topic.
package producer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	kafkautil "github.com/afikmenashe/alerting-platform/pkg/kafka"
	protorules "github.com/afikmenashe/alerting-platform/pkg/proto/rules"
	"rule-service/internal/events"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

// Producer wraps a Kafka writer and provides a simple interface for publishing rule changed events.
type Producer struct {
	writer *kafka.Writer
	topic  string
}

// NewProducer creates a new Kafka producer with the specified brokers and topic.
// The producer is configured for at-least-once delivery semantics with synchronous writes.
func NewProducer(brokers string, topic string) (*Producer, error) {
	if err := kafkautil.ValidateProducerParams(brokers, topic); err != nil {
		return nil, err
	}

	// Parse comma-separated broker list
	brokerList := kafkautil.ParseBrokers(brokers)

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
		WriteTimeout: kafkautil.WriteTimeout,
		RequiredAcks: kafka.RequireOne, // At-least-once semantics (waits for leader ack)
		Async:        false,            // Synchronous writes for reliability and error handling
		BatchSize:    1,                // Flush immediately, no batching delay
	}

	slog.Info("Kafka producer configured",
		"write_timeout", kafkautil.WriteTimeout,
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



// Publish serializes a rule changed event to protobuf and publishes it to Kafka.
// The message is keyed by rule_id for partition distribution.
// Returns an error if serialization or publishing fails.
func (p *Producer) Publish(ctx context.Context, changed *events.RuleChanged) error {
	evt := &protorules.RuleChanged{
		RuleId:        changed.RuleID,
		ClientId:      changed.ClientID,
		Action:        events.ToProtoAction(changed.Action),
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
