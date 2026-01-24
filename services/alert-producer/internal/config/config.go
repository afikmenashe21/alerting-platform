// Package config provides configuration parsing and validation for the alert-producer service.
// It handles parsing of distribution strings and validates configuration parameters.
package config

import (
	"fmt"
	"strings"
	"time"
)

// Config holds all configuration parameters for the alert-producer service.
type Config struct {
	KafkaBrokers string
	Topic        string
	RPS          float64
	Duration     time.Duration
	BurstSize    int
	Seed         int64
	SeverityDist string
	SourceDist   string
	NameDist     string
	RedisAddr    string
}

// Validate checks that all required configuration fields are set and have valid values.
// It also validates that distribution strings are properly formatted and sum to 100.
// Returns an error if validation fails, nil otherwise.
func (c *Config) Validate() error {
	if c.KafkaBrokers == "" {
		return fmt.Errorf("kafka-brokers cannot be empty")
	}
	if c.Topic == "" {
		return fmt.Errorf("topic cannot be empty")
	}
	if c.RPS <= 0 && c.BurstSize <= 0 {
		return fmt.Errorf("rps must be > 0 or burst must be > 0")
	}
	if c.BurstSize == 0 && c.Duration <= 0 {
		return fmt.Errorf("duration must be > 0 when not in burst mode")
	}
	
	// Validate distribution strings early to provide better error messages
	if _, err := ParseDistribution(c.SeverityDist); err != nil {
		return fmt.Errorf("invalid severity-dist: %w", err)
	}
	if _, err := ParseDistribution(c.SourceDist); err != nil {
		return fmt.Errorf("invalid source-dist: %w", err)
	}
	if _, err := ParseDistribution(c.NameDist); err != nil {
		return fmt.Errorf("invalid name-dist: %w", err)
	}
	
	return nil
}

// ParseDistribution parses a weighted distribution string into a map of values to percentages.
//
// Format: "KEY1:PERCENT1,KEY2:PERCENT2,..." where percentages must sum to 100.
//
// Example: "HIGH:30,MEDIUM:40,LOW:20,CRITICAL:10"
//
// Returns a map of value -> percentage (0-100) and an error if parsing fails.
func ParseDistribution(distStr string) (map[string]int, error) {
	result := make(map[string]int)
	
	if distStr == "" {
		return result, fmt.Errorf("distribution string cannot be empty")
	}
	
	parts := strings.Split(distStr, ",")
	totalPercent := 0
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		
		kv := strings.Split(part, ":")
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid distribution format: %s (expected KEY:PERCENT)", part)
		}
		
		key := strings.TrimSpace(kv[0])
		var percent int
		if _, err := fmt.Sscanf(kv[1], "%d", &percent); err != nil {
			return nil, fmt.Errorf("invalid percentage in %s: %w", part, err)
		}
		
		if percent < 0 || percent > 100 {
			return nil, fmt.Errorf("percentage must be 0-100, got %d in %s", percent, part)
		}
		
		result[key] = percent
		totalPercent += percent
	}
	
	if totalPercent != 100 {
		return nil, fmt.Errorf("distribution percentages must sum to 100, got %d", totalPercent)
	}
	
	return result, nil
}
