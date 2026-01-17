// Package config provides configuration parsing and validation for the rule-service.
package config

import (
	"fmt"
)

// Config holds all configuration parameters for the rule-service.
type Config struct {
	HTTPPort        string
	KafkaBrokers    string
	RuleChangedTopic string
	PostgresDSN     string
}

// Validate checks that all required configuration fields are set and have valid values.
// Returns an error if validation fails, nil otherwise.
func (c *Config) Validate() error {
	if c.HTTPPort == "" {
		return fmt.Errorf("http-port cannot be empty")
	}
	if c.KafkaBrokers == "" {
		return fmt.Errorf("kafka-brokers cannot be empty")
	}
	if c.RuleChangedTopic == "" {
		return fmt.Errorf("rule-changed-topic cannot be empty")
	}
	if c.PostgresDSN == "" {
		return fmt.Errorf("postgres-dsn cannot be empty")
	}
	return nil
}
