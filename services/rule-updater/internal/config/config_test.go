package config

import (
	"testing"
)

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
				KafkaBrokers:     "localhost:9092",
				RuleChangedTopic: "rule.changed",
				ConsumerGroupID:  "rule-updater-group",
				PostgresDSN:       "postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable",
				RedisAddr:         "localhost:6379",
			},
			wantErr: false,
		},
		{
			name: "missing kafka-brokers",
			config: Config{
				RuleChangedTopic: "rule.changed",
				ConsumerGroupID:  "rule-updater-group",
				PostgresDSN:       "postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable",
				RedisAddr:         "localhost:6379",
			},
			wantErr: true,
			errMsg:  "kafka-brokers cannot be empty",
		},
		{
			name: "missing rule-changed-topic",
			config: Config{
				KafkaBrokers:    "localhost:9092",
				ConsumerGroupID: "rule-updater-group",
				PostgresDSN:     "postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable",
				RedisAddr:       "localhost:6379",
			},
			wantErr: true,
			errMsg:  "rule-changed-topic cannot be empty",
		},
		{
			name: "missing consumer-group-id",
			config: Config{
				KafkaBrokers:     "localhost:9092",
				RuleChangedTopic: "rule.changed",
				PostgresDSN:      "postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable",
				RedisAddr:        "localhost:6379",
			},
			wantErr: true,
			errMsg:  "consumer-group-id cannot be empty",
		},
		{
			name: "missing postgres-dsn",
			config: Config{
				KafkaBrokers:     "localhost:9092",
				RuleChangedTopic: "rule.changed",
				ConsumerGroupID:  "rule-updater-group",
				RedisAddr:        "localhost:6379",
			},
			wantErr: true,
			errMsg:  "postgres-dsn cannot be empty",
		},
		{
			name: "missing redis-addr",
			config: Config{
				KafkaBrokers:     "localhost:9092",
				RuleChangedTopic: "rule.changed",
				ConsumerGroupID:  "rule-updater-group",
				PostgresDSN:       "postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable",
			},
			wantErr: true,
			errMsg:  "redis-addr cannot be empty",
		},
		{
			name: "all fields empty",
			config: Config{
				KafkaBrokers:     "",
				RuleChangedTopic: "",
				ConsumerGroupID:  "",
				PostgresDSN:      "",
				RedisAddr:        "",
			},
			wantErr: true,
			errMsg:  "kafka-brokers cannot be empty",
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
