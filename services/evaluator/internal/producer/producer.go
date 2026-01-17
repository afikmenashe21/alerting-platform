// Package producer provides Kafka producer functionality for alerts.matched topic.
package producer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"evaluator/internal/events"
	"github.com/segmentio/kafka-go"
)

const (
	// writeTimeout is the maximum time to wait for a Kafka write operation.
	writeTimeout = 10 * time.Second
)

// Producer wraps a Kafka writer and provides a simple interface for publishing matched alerts.
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
	// Use Hash balancer to partition by client_id for tenant locality
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
		"partition_key", "client_id (hashed)",
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
		NumPartitions:     3,
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
}

// Publish serializes a matched alert to JSON and publishes it to Kafka.
// The message is keyed by client_id for partition distribution (tenant locality).
// Returns an error if serialization or publishing fails.
func (p *Producer) Publish(ctx context.Context, matched *events.AlertMatched) error {
	// Serialize matched alert to JSON
	payload, err := json.Marshal(matched)
	if err != nil {
		slog.Error("Failed to marshal matched alert to JSON",
			"alert_id", matched.AlertID,
			"client_id", matched.ClientID,
			"error", err,
		)
		return fmt.Errorf("failed to marshal matched alert: %w", err)
	}

	// Partition key: use client_id for tenant locality
	partitionKey := []byte(matched.ClientID)

	// Create Kafka message with key, value, headers, and timestamp
	msg := kafka.Message{
		Key:   partitionKey,
		Value: payload,
		Headers: []kafka.Header{
			{
				Key:   "schema_version",
				Value: []byte(fmt.Sprintf("%d", matched.SchemaVersion)),
			},
			{
				Key:   "alert_id",
				Value: []byte(matched.AlertID),
			},
		},
		Time: time.Unix(matched.EventTS, 0), // Set message timestamp from alert
	}

	// Write to Kafka (synchronous, waits for ack)
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		slog.Error("Failed to write message to Kafka",
			"alert_id", matched.AlertID,
			"topic", p.topic,
			"error", err,
		)
		return fmt.Errorf("failed to write message to Kafka: %w", err)
	}

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
