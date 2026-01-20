// Package producer provides a Kafka producer wrapper for publishing alerts.
// It handles message serialization, keying, and Kafka-specific configuration.
package producer

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"alert-producer/internal/generator"
	"github.com/segmentio/kafka-go"
)

const (
	// writeTimeout is the maximum time to wait for a Kafka write operation
	writeTimeout = 10 * time.Second
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
	// Use Hash balancer to partition by key (alert_id) for even distribution
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokerList...),
		Topic:        topic,
		Balancer:     &kafka.Hash{}, // Key-based partitioning (hashes the message key)
		WriteTimeout: writeTimeout,
		RequiredAcks: kafka.RequireOne,   // At-least-once semantics (waits for leader ack)
		Async:        false,              // Synchronous writes for reliability and error handling
	}
	
	slog.Info("Kafka producer configured",
		"write_timeout", writeTimeout,
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

// createTopicIfNotExists attempts to create the topic if it doesn't exist.
// This is a best-effort operation and failures are logged but don't prevent producer creation.
func createTopicIfNotExists(broker, topic string) {
	conn, err := kafka.Dial("tcp", broker)
	if err != nil {
		slog.Warn("Could not connect to Kafka to check/create topic",
			"broker", broker,
			"topic", topic,
			"error", err,
			"note", "Topic may need to be created manually",
		)
		return
	}
	defer conn.Close()

	// Check if topic exists
	partitions, err := conn.ReadPartitions(topic)
	if err == nil && len(partitions) > 0 {
		slog.Info("Topic already exists",
			"topic", topic,
			"partitions", len(partitions),
		)
		return
	}

	// Topic doesn't exist, try to create it
	topicConfig := kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:    3,
		ReplicationFactor: 1,
	}

	err = conn.CreateTopics(topicConfig)
	if err != nil {
		slog.Warn("Could not create topic (may need to be created manually)",
			"topic", topic,
			"error", err,
			"tip", "Run: docker exec kafka kafka-topics --create --bootstrap-server localhost:9092 --topic "+topic+" --partitions 3 --replication-factor 1",
		)
		return
	}

	slog.Info("Created topic",
		"topic", topic,
		"partitions", 3,
		"replication_factor", 1,
	)

	// Wait for topic to be fully available (Kafka topic creation is asynchronous)
	// Retry reading partitions up to 5 times with 1 second delay
	maxRetries := 5
	retryDelay := 1 * time.Second
	for i := 0; i < maxRetries; i++ {
		time.Sleep(retryDelay)
		partitions, err := conn.ReadPartitions(topic)
		if err == nil && len(partitions) > 0 {
			slog.Info("Topic is now available",
				"topic", topic,
				"partitions", len(partitions),
			)
			return
		}
		if i < maxRetries-1 {
			slog.Info("Waiting for topic to be available",
				"topic", topic,
				"attempt", i+1,
				"max_retries", maxRetries,
			)
		}
	}

	// If we still can't read partitions, try one more time with a fresh connection
	// This handles the case where the connection was stale
	verifyConn, err := kafka.Dial("tcp", broker)
	if err == nil {
		defer verifyConn.Close()
		partitions, err := verifyConn.ReadPartitions(topic)
		if err == nil && len(partitions) > 0 {
			slog.Info("Topic is now available (verified with new connection)",
				"topic", topic,
				"partitions", len(partitions),
			)
			return
		}
	}

	slog.Warn("Topic created but may not be fully available yet",
		"topic", topic,
		"note", "Producer will retry on first write if topic is not ready",
	)
}

// Publish serializes an alert to JSON and publishes it to Kafka.
// The message is keyed by alert_id for even partition distribution.
// Returns an error if serialization or publishing fails.
func (p *Producer) Publish(ctx context.Context, alert *generator.Alert) error {
	// Serialize alert to JSON
	payload, err := json.Marshal(alert)
	if err != nil {
		slog.Error("Failed to marshal alert to JSON",
			"alert_id", alert.AlertID,
			"error", err,
		)
		return fmt.Errorf("failed to marshal alert: %w", err)
	}
	
	// Create Kafka message with key, value, headers, and timestamp
	// Partition key: hash of alert_id for even distribution across partitions
	// - Prevents hot partitions by ensuring random distribution
	// - Same alert_id always maps to same partition (deterministic)
	// - Kafka's Hash balancer will hash this key again internally
	partitionKey := hashAlertID(alert.AlertID)
	msg := kafka.Message{
		Key:   partitionKey,
		Value: payload,
		Headers: []kafka.Header{
			{
				Key:   "schema_version",
				Value: []byte(fmt.Sprintf("%d", alert.SchemaVersion)),
			},
			{
				Key:   "severity",
				Value: []byte(alert.Severity),
			},
		},
		Time: time.Unix(alert.EventTS, 0), // Set message timestamp from alert
	}
	
	// Write to Kafka (synchronous, waits for ack)
	// Retry once if topic is not ready (handles async topic creation)
	maxRetries := 2
	var writeErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Check if context is cancelled before attempting to write
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		writeErr = p.writer.WriteMessages(ctx, msg)
		if writeErr == nil {
			return nil
		}

		// Check if error is due to context cancellation
		if errors.Is(writeErr, context.Canceled) || ctx.Err() == context.Canceled {
			return context.Canceled
		}

		// Check if error is due to topic not existing
		errStr := writeErr.Error()
		if (strings.Contains(errStr, "Unknown Topic Or Partition") || 
			strings.Contains(errStr, "does not exist")) && attempt < maxRetries {
			slog.Info("Topic not ready, retrying after delay",
				"alert_id", alert.AlertID,
				"topic", p.topic,
				"attempt", attempt,
				"max_retries", maxRetries,
			)
			time.Sleep(2 * time.Second)
			continue
		}

		// For other errors or final attempt, return error
		slog.Error("Failed to write message to Kafka",
			"alert_id", alert.AlertID,
			"topic", p.topic,
			"error", writeErr,
			"attempt", attempt,
		)
		return fmt.Errorf("failed to write message to Kafka: %w", writeErr)
	}

	return fmt.Errorf("failed to write message to Kafka after %d attempts: %w", maxRetries, writeErr)
}

// hashAlertID creates a deterministic hash of the alert_id for partition key.
// This ensures even distribution across partitions and avoids hot partitions.
// The hash is deterministic so the same alert_id always maps to the same partition.
// Note: Kafka's Hash balancer will hash this key again internally, but pre-hashing
// gives us explicit control and ensures good distribution even if Kafka's hashing changes.
func hashAlertID(alertID string) []byte {
	hash := sha256.Sum256([]byte(alertID))
	// Return first 16 bytes for efficiency (Kafka will hash this again internally)
	// This provides good distribution while keeping the key size reasonable
	return hash[:16]
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
