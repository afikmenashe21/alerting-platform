// Package config provides tests for configuration validation.
package config

import (
	"testing"
)

// TestConfig_Validate tests the Validate method with various scenarios.
func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				HTTPPort:         "8081",
				KafkaBrokers:     "localhost:9092",
				RuleChangedTopic: "rule.changed",
				PostgresDSN:      "postgres://user:pass@localhost:5432/db",
			},
			wantErr: false,
		},
		{
			name: "empty http-port",
			config: Config{
				HTTPPort:         "",
				KafkaBrokers:     "localhost:9092",
				RuleChangedTopic: "rule.changed",
				PostgresDSN:      "postgres://user:pass@localhost:5432/db",
			},
			wantErr: true,
			errMsg:  "http-port cannot be empty",
		},
		{
			name: "empty kafka-brokers",
			config: Config{
				HTTPPort:         "8081",
				KafkaBrokers:     "",
				RuleChangedTopic: "rule.changed",
				PostgresDSN:      "postgres://user:pass@localhost:5432/db",
			},
			wantErr: true,
			errMsg:  "kafka-brokers cannot be empty",
		},
		{
			name: "empty rule-changed-topic",
			config: Config{
				HTTPPort:         "8081",
				KafkaBrokers:     "localhost:9092",
				RuleChangedTopic: "",
				PostgresDSN:      "postgres://user:pass@localhost:5432/db",
			},
			wantErr: true,
			errMsg:  "rule-changed-topic cannot be empty",
		},
		{
			name: "empty postgres-dsn",
			config: Config{
				HTTPPort:         "8081",
				KafkaBrokers:     "localhost:9092",
				RuleChangedTopic: "rule.changed",
				PostgresDSN:      "",
			},
			wantErr: true,
			errMsg:  "postgres-dsn cannot be empty",
		},
		{
			name: "all fields empty",
			config: Config{
				HTTPPort:         "",
				KafkaBrokers:     "",
				RuleChangedTopic: "",
				PostgresDSN:      "",
			},
			wantErr: true,
			errMsg:  "http-port cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if err.Error() != tt.errMsg {
					t.Errorf("Config.Validate() error = %v, want error message %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}
