// Package ruleconsumer provides Kafka consumer functionality for rule.changed topic.
package ruleconsumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"evaluator/internal/events"
	kafkautil "evaluator/internal/kafka"
	"github.com/segmentio/kafka-go"
)

// Consumer wraps a Kafka reader for consuming rule.changed events.
type Consumer struct {
	reader *kafka.Reader
	topic  string
}

// NewConsumer creates a new Kafka consumer for rule.changed topic.
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
	brokerList := kafkautil.ParseBrokers(brokers)

	slog.Info("Initializing rule.changed Kafka consumer",
		"brokers", brokerList,
		"topic", topic,
		"group_id", groupID,
	)

	// Configure Kafka reader for at-least-once delivery
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokerList,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		MaxWait:        kafkautil.ReadTimeout,
		CommitInterval: kafkautil.CommitInterval,
		StartOffset:    kafka.FirstOffset, // Start from beginning if no committed offset
	})

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

	var ruleChanged events.RuleChanged
	if err := json.Unmarshal(msg.Value, &ruleChanged); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rule changed event: %w", err)
	}

	return &ruleChanged, nil
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
