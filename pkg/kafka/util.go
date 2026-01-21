// Package kafka provides shared Kafka utilities for all services.
package kafka

import (
	"fmt"
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

// NewReaderConfig creates a standard Kafka reader configuration for at-least-once delivery.
// This configuration is shared across all consumers in the platform.
func NewReaderConfig(brokers []string, topic, groupID string) kafka.ReaderConfig {
	return kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		MaxWait:        ReadTimeout,
		CommitInterval: CommitInterval,
		StartOffset:    kafka.FirstOffset, // Start from beginning if no committed offset
	}
}
