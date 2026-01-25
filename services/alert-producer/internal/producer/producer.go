// Package producer provides a Kafka producer wrapper for publishing alerts.
// It handles message serialization, keying, and Kafka-specific configuration.
package producer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"alert-producer/internal/generator"

	kafkautil "github.com/afikmenashe/alerting-platform/pkg/kafka"
	"github.com/segmentio/kafka-go"
)

// AlertPublisher defines the interface for publishing alerts.
type AlertPublisher interface {
	Publish(ctx context.Context, alert *generator.Alert) error
	Close() error
}

// Producer wraps a Kafka writer and provides a simple interface for publishing alerts.
// Messages are keyed by alert_id for even distribution across partitions.
type Producer struct {
	writer *kafka.Writer
	topic  string
}

// Ensure Producer implements AlertPublisher interface
var _ AlertPublisher = (*Producer)(nil)

// New creates a new Kafka producer with the specified brokers and topic.
// The producer is configured for at-least-once delivery semantics with synchronous writes.
// It will attempt to create the topic if it doesn't exist (with 3 partitions, replication factor 1).
func New(brokers string, topic string) (*Producer, error) {
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
	// Use Hash balancer to partition by key (alert_id) for even distribution
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
		"partition_key", "alert_id (hashed)",
	)

	return &Producer{
		writer: writer,
		topic:  topic,
	}, nil
}


const (
	// maxWriteRetries is the number of attempts for writing to Kafka.
	maxWriteRetries = 2
	// retryDelay is the delay between retries when topic is not ready.
	retryDelay = 2 * time.Second
)

// Publish serializes an alert to protobuf and publishes it to Kafka.
// The message is keyed by alert_id for even partition distribution.
// Returns an error if serialization or publishing fails.
func (p *Producer) Publish(ctx context.Context, alert *generator.Alert) error {
	payload, err := encodeAlert(alert)
	if err != nil {
		slog.Error("Failed to marshal alert to protobuf",
			"alert_id", alert.AlertID,
			"error", err,
		)
		return err
	}

	msg := buildKafkaMessage(alert, payload)
	return p.writeWithRetry(ctx, msg, alert.AlertID)
}

// writeWithRetry writes a message to Kafka with retry logic for transient errors.
// Retries once if the topic is not ready (handles async topic creation).
func (p *Producer) writeWithRetry(ctx context.Context, msg kafka.Message, alertID string) error {
	var writeErr error

	for attempt := 1; attempt <= maxWriteRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		writeErr = p.writer.WriteMessages(ctx, msg)
		if writeErr == nil {
			return nil
		}

		if errors.Is(writeErr, context.Canceled) || ctx.Err() == context.Canceled {
			return context.Canceled
		}

		if isTopicNotReadyError(writeErr) && attempt < maxWriteRetries {
			slog.Info("Topic not ready, retrying after delay",
				"alert_id", alertID,
				"topic", p.topic,
				"attempt", attempt,
			)
			time.Sleep(retryDelay)
			continue
		}

		slog.Error("Failed to write message to Kafka",
			"alert_id", alertID,
			"topic", p.topic,
			"error", writeErr,
			"attempt", attempt,
		)
		return fmt.Errorf("failed to write message to Kafka: %w", writeErr)
	}

	return fmt.Errorf("failed to write message to Kafka after %d attempts: %w", maxWriteRetries, writeErr)
}

// isTopicNotReadyError checks if the error indicates the topic doesn't exist yet.
func isTopicNotReadyError(err error) bool {
	errStr := err.Error()
	return strings.Contains(errStr, "Unknown Topic Or Partition") ||
		strings.Contains(errStr, "does not exist")
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
