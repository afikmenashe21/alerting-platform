// Package kafka provides shared Kafka utilities for all services.
package kafka

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/segmentio/kafka-go"
)

// ParseBrokers parses a comma-separated broker list and trims whitespace.
// Returns a slice of broker addresses.
func ParseBrokers(brokers string) []string {
	if brokers == "" {
		return nil
	}
	brokerList := strings.Split(brokers, ",")
	for i := range brokerList {
		brokerList[i] = strings.TrimSpace(brokerList[i])
	}
	return brokerList
}

// ValidateConsumerParams validates common consumer parameters.
// Returns an error if any parameter is invalid.
func ValidateConsumerParams(brokers, topic, groupID string) error {
	if brokers == "" {
		return fmt.Errorf("brokers cannot be empty")
	}
	if topic == "" {
		return fmt.Errorf("topic cannot be empty")
	}
	if groupID == "" {
		return fmt.Errorf("groupID cannot be empty")
	}
	return nil
}

// ValidateProducerParams validates common producer parameters.
// Returns an error if any parameter is invalid.
func ValidateProducerParams(brokers, topic string) error {
	if brokers == "" {
		return fmt.Errorf("brokers cannot be empty")
	}
	if topic == "" {
		return fmt.Errorf("topic cannot be empty")
	}
	return nil
}

// ReaderConfigValues holds the actual values used in the reader config for logging.
type ReaderConfigValues struct {
	MinBytes       int
	MaxBytes       int
	MaxWait        string
	CommitInterval string
}

// GetReaderConfigValues returns the actual configuration values for logging purposes.
// This ensures services log the correct centralized values.
func GetReaderConfigValues() ReaderConfigValues {
	return ReaderConfigValues{
		MinBytes:       1,
		MaxBytes:       10e6,
		MaxWait:        MaxPollWait.String(),
		CommitInterval: CommitInterval.String(),
	}
}

// LogReaderConfig logs the reader configuration values.
// Call this after creating a reader to log the actual config being used.
func LogReaderConfig() {
	cfg := GetReaderConfigValues()
	slog.Info("Kafka consumer configured",
		"min_bytes", cfg.MinBytes,
		"max_bytes", cfg.MaxBytes,
		"max_wait", cfg.MaxWait,
		"commit_interval", cfg.CommitInterval,
	)
}

// NewReaderConfig creates a standard Kafka reader configuration for at-least-once delivery.
// This configuration is shared across all consumers in the platform.
func NewReaderConfig(brokers []string, topic, groupID string) kafka.ReaderConfig {
	return kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       1,    // Return immediately when any data is available
		MaxBytes:       10e6, // 10MB
		MaxWait:        MaxPollWait,
		CommitInterval: CommitInterval,
		StartOffset:    kafka.FirstOffset, // Start from beginning if no committed offset
	}
}
