// Package config provides configuration parsing and validation for the rule-updater service.
package config

import (
	"fmt"
)

// Config holds all configuration parameters for the rule-updater service.
type Config struct {
	KafkaBrokers      string
	RuleChangedTopic  string
	ConsumerGroupID   string
	PostgresDSN       string
	RedisAddr         string
}

// Validate checks that all required configuration fields are set and have valid values.
// Returns an error if validation fails, nil otherwise.
func (c *Config) Validate() error {
	if c.KafkaBrokers == "" {
		return fmt.Errorf("kafka-brokers cannot be empty")
	}
	if c.RuleChangedTopic == "" {
		return fmt.Errorf("rule-changed-topic cannot be empty")
	}
	if c.ConsumerGroupID == "" {
		return fmt.Errorf("consumer-group-id cannot be empty")
	}
	if c.PostgresDSN == "" {
		return fmt.Errorf("postgres-dsn cannot be empty")
	}
	if c.RedisAddr == "" {
		return fmt.Errorf("redis-addr cannot be empty")
	}
	return nil
}
