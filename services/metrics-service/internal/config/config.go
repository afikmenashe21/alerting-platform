// Package config provides configuration parsing and validation for the metrics-service.
package config

import (
	"fmt"
)

// Config holds all configuration parameters for the metrics-service.
type Config struct {
	HTTPPort    string
	PostgresDSN string
	RedisAddr   string
}

// Validate checks that all required configuration fields are set and have valid values.
func (c *Config) Validate() error {
	if c.HTTPPort == "" {
		return fmt.Errorf("http-port cannot be empty")
	}
	if c.PostgresDSN == "" {
		return fmt.Errorf("postgres-dsn cannot be empty")
	}
	if c.RedisAddr == "" {
		return fmt.Errorf("redis-addr cannot be empty")
	}
	return nil
}
