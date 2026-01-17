// Package consumer provides Kafka consumer functionality for rule.changed topic.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"rule-updater/internal/events"
	"github.com/segmentio/kafka-go"
)

const (
	// readTimeout is the maximum time to wait for a Kafka read operation.
	readTimeout = 10 * time.Second
	// commitInterval is how often to commit offsets (after processing).
	commitInterval = 1 * time.Second
)

// Consumer wraps a Kafka reader and provides a simple interface for consuming rule.changed events.
type Consumer struct {
	reader *kafka.Reader
	topic  string
}

// NewConsumer creates a new Kafka consumer with the specified brokers, topic, and group ID.
// The consumer is configured for at-least-once delivery semantics.
func NewConsumer(brokers string, topic string, groupID string) (*Consumer, error) {
	if brokers == "" {
		return nil, fmt.Errorf("brokers cannot be empty")
	}
	if topic == "" {
		return nil, fmt.Errorf("topic cannot be empty")
	}
	if groupID == "" {
		return nil, fmt.Errorf("groupID cannot be empty")
	}

	// Parse comma-separated broker list
	brokerList := strings.Split(brokers, ",")
	for i := range brokerList {
		brokerList[i] = strings.TrimSpace(brokerList[i])
	}

	slog.Info("Initializing Kafka consumer",
		"brokers", brokerList,
		"topic", topic,
		"group_id", groupID,
	)

	// Configure Kafka reader for at-least-once delivery
	// StartOffset only applies when no committed offset exists for the consumer group
	// Using FirstOffset ensures we read all messages when starting fresh
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokerList,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		MaxWait:        readTimeout,
		CommitInterval: commitInterval,
		StartOffset:    kafka.FirstOffset, // Start from beginning if no committed offset
	})

	slog.Info("Kafka consumer configured",
		"min_bytes", 10e3,
		"max_bytes", 10e6,
		"max_wait", readTimeout,
		"commit_interval", commitInterval,
	)

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

	var ruleChanged events.RuleChanged
	if err := json.Unmarshal(msg.Value, &ruleChanged); err != nil {
		return nil, &msg, fmt.Errorf("failed to unmarshal rule.changed event: %w", err)
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
