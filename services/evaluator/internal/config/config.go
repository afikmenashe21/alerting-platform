// Package config provides configuration parsing and validation for the evaluator service.
package config

import (
	"fmt"
	"time"
)

// Config holds all configuration parameters for the evaluator service.
type Config struct {
	KafkaBrokers        string
	AlertsNewTopic      string
	AlertsMatchedTopic  string
	RuleChangedTopic    string
	ConsumerGroupID     string
	RuleChangedGroupID  string
	RedisAddr           string
	VersionPollInterval time.Duration
}

// Validate checks that all required configuration fields are set and have valid values.
// Returns an error if validation fails, nil otherwise.
func (c *Config) Validate() error {
	if c.KafkaBrokers == "" {
		return fmt.Errorf("kafka-brokers cannot be empty")
	}
	if c.AlertsNewTopic == "" {
		return fmt.Errorf("alerts-new-topic cannot be empty")
	}
	if c.AlertsMatchedTopic == "" {
		return fmt.Errorf("alerts-matched-topic cannot be empty")
	}
	if c.ConsumerGroupID == "" {
		return fmt.Errorf("consumer-group-id cannot be empty")
	}
	if c.RuleChangedTopic == "" {
		return fmt.Errorf("rule-changed-topic cannot be empty")
	}
	if c.RuleChangedGroupID == "" {
		return fmt.Errorf("rule-changed-group-id cannot be empty")
	}
	if c.RedisAddr == "" {
		return fmt.Errorf("redis-addr cannot be empty")
	}
	if c.VersionPollInterval <= 0 {
		return fmt.Errorf("version-poll-interval must be > 0")
	}
	return nil
}
