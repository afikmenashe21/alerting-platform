// Package config provides configuration parsing and validation for the sender service.
package config

import (
	"fmt"
)

// Config holds all configuration parameters for the sender service.
type Config struct {
	KafkaBrokers            string
	NotificationsReadyTopic string
	ConsumerGroupID         string
	PostgresDSN             string
}

// Validate checks that all required configuration fields are set and have valid values.
// Returns an error if validation fails, nil otherwise.
func (c *Config) Validate() error {
	if c.KafkaBrokers == "" {
		return fmt.Errorf("kafka-brokers cannot be empty")
	}
	if c.NotificationsReadyTopic == "" {
		return fmt.Errorf("notifications-ready-topic cannot be empty")
	}
	if c.ConsumerGroupID == "" {
		return fmt.Errorf("consumer-group-id cannot be empty")
	}
	if c.PostgresDSN == "" {
		return fmt.Errorf("postgres-dsn cannot be empty")
	}
	return nil
}
