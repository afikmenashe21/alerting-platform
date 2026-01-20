// Package producer provides Kafka producer functionality for notifications.ready topic.
package producer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"aggregator/internal/events"
	kafkautil "aggregator/internal/kafka"

	"github.com/segmentio/kafka-go"
)

// Producer wraps a Kafka writer and provides a simple interface for publishing notification ready events.
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

	// Configure Kafka writer for at-least-once delivery
	// Use Hash balancer to partition by client_id for tenant locality
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokerList...),
		Topic:        topic,
		Balancer:     &kafka.Hash{}, // Key-based partitioning (hashes the message key)
		WriteTimeout: kafkautil.WriteTimeout,
		RequiredAcks: kafka.RequireOne, // At-least-once semantics (waits for leader ack)
		Async:        false,            // Synchronous writes for reliability and error handling
	}

	slog.Info("Kafka producer configured",
		"write_timeout", kafkautil.WriteTimeout,
		"required_acks", "RequireOne",
		"async", false,
		"balancer", "Hash (key-based partitioning)",
		"partition_key", "client_id (hashed)",
	)

	return &Producer{
		writer: writer,
		topic:  topic,
	}, nil
}

// buildMessage creates a Kafka message from a NotificationReady event.
// The message is keyed by client_id for partition distribution (tenant locality).
func buildMessage(ready *events.NotificationReady) (kafka.Message, error) {
	// Serialize notification ready event to JSON
	payload, err := json.Marshal(ready)
	if err != nil {
		return kafka.Message{}, fmt.Errorf("failed to marshal notification ready event: %w", err)
	}

	// Partition key: use client_id for tenant locality
	partitionKey := []byte(ready.ClientID)

	// Create Kafka message with key, value, headers, and timestamp
	msg := kafka.Message{
		Key:   partitionKey,
		Value: payload,
		Headers: []kafka.Header{
			{
				Key:   "schema_version",
				Value: []byte(fmt.Sprintf("%d", ready.SchemaVersion)),
			},
			{
				Key:   "notification_id",
				Value: []byte(ready.NotificationID),
			},
		},
		Time: time.Now(),
	}

	return msg, nil
}

// Publish serializes a notification ready event to JSON and publishes it to Kafka.
// The message is keyed by client_id for partition distribution (tenant locality).
// Returns an error if serialization or publishing fails.
func (p *Producer) Publish(ctx context.Context, ready *events.NotificationReady) error {
	// Build Kafka message
	msg, err := buildMessage(ready)
	if err != nil {
		slog.Error("Failed to build notification ready message",
			"notification_id", ready.NotificationID,
			"client_id", ready.ClientID,
			"error", err,
		)
		return err
	}

	// Write to Kafka (synchronous, waits for ack)
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		slog.Error("Failed to write message to Kafka",
			"notification_id", ready.NotificationID,
			"topic", p.topic,
			"error", err,
		)
		return fmt.Errorf("failed to write message to Kafka: %w", err)
	}

	slog.Info("Published notification ready event",
		"notification_id", ready.NotificationID,
		"client_id", ready.ClientID,
		"alert_id", ready.AlertID,
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
