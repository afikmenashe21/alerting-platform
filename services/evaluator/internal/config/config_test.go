package config

import (
	"testing"
	"time"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				KafkaBrokers:        "localhost:9092",
				AlertsNewTopic:      "alerts.new",
				AlertsMatchedTopic:  "alerts.matched",
				RuleChangedTopic:    "rule.changed",
				ConsumerGroupID:     "evaluator-group",
				RuleChangedGroupID:  "evaluator-rule-changed-group",
				RedisAddr:           "localhost:6379",
				VersionPollInterval: 5 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "empty kafka brokers",
			config: &Config{
				KafkaBrokers:        "",
				AlertsNewTopic:      "alerts.new",
				AlertsMatchedTopic:  "alerts.matched",
				RuleChangedTopic:    "rule.changed",
				ConsumerGroupID:     "evaluator-group",
				RuleChangedGroupID:  "evaluator-rule-changed-group",
				RedisAddr:           "localhost:6379",
				VersionPollInterval: 5 * time.Second,
			},
			wantErr: true,
			errMsg:  "kafka-brokers cannot be empty",
		},
		{
			name: "empty alerts new topic",
			config: &Config{
				KafkaBrokers:        "localhost:9092",
				AlertsNewTopic:      "",
				AlertsMatchedTopic:  "alerts.matched",
				RuleChangedTopic:    "rule.changed",
				ConsumerGroupID:     "evaluator-group",
				RuleChangedGroupID:  "evaluator-rule-changed-group",
				RedisAddr:           "localhost:6379",
				VersionPollInterval: 5 * time.Second,
			},
			wantErr: true,
			errMsg:  "alerts-new-topic cannot be empty",
		},
		{
			name: "empty alerts matched topic",
			config: &Config{
				KafkaBrokers:        "localhost:9092",
				AlertsNewTopic:      "alerts.new",
				AlertsMatchedTopic:  "",
				RuleChangedTopic:    "rule.changed",
				ConsumerGroupID:     "evaluator-group",
				RuleChangedGroupID:  "evaluator-rule-changed-group",
				RedisAddr:           "localhost:6379",
				VersionPollInterval: 5 * time.Second,
			},
			wantErr: true,
			errMsg:  "alerts-matched-topic cannot be empty",
		},
		{
			name: "empty consumer group id",
			config: &Config{
				KafkaBrokers:        "localhost:9092",
				AlertsNewTopic:      "alerts.new",
				AlertsMatchedTopic:  "alerts.matched",
				RuleChangedTopic:    "rule.changed",
				ConsumerGroupID:     "",
				RuleChangedGroupID:  "evaluator-rule-changed-group",
				RedisAddr:           "localhost:6379",
				VersionPollInterval: 5 * time.Second,
			},
			wantErr: true,
			errMsg:  "consumer-group-id cannot be empty",
		},
		{
			name: "empty rule changed topic",
			config: &Config{
				KafkaBrokers:        "localhost:9092",
				AlertsNewTopic:      "alerts.new",
				AlertsMatchedTopic:  "alerts.matched",
				RuleChangedTopic:    "",
				ConsumerGroupID:     "evaluator-group",
				RuleChangedGroupID:  "evaluator-rule-changed-group",
				RedisAddr:           "localhost:6379",
				VersionPollInterval: 5 * time.Second,
			},
			wantErr: true,
			errMsg:  "rule-changed-topic cannot be empty",
		},
		{
			name: "empty rule changed group id",
			config: &Config{
				KafkaBrokers:        "localhost:9092",
				AlertsNewTopic:      "alerts.new",
				AlertsMatchedTopic:  "alerts.matched",
				RuleChangedTopic:    "rule.changed",
				ConsumerGroupID:     "evaluator-group",
				RuleChangedGroupID:  "",
				RedisAddr:           "localhost:6379",
				VersionPollInterval: 5 * time.Second,
			},
			wantErr: true,
			errMsg:  "rule-changed-group-id cannot be empty",
		},
		{
			name: "empty redis addr",
			config: &Config{
				KafkaBrokers:        "localhost:9092",
				AlertsNewTopic:      "alerts.new",
				AlertsMatchedTopic:  "alerts.matched",
				RuleChangedTopic:    "rule.changed",
				ConsumerGroupID:     "evaluator-group",
				RuleChangedGroupID:  "evaluator-rule-changed-group",
				RedisAddr:           "",
				VersionPollInterval: 5 * time.Second,
			},
			wantErr: true,
			errMsg:  "redis-addr cannot be empty",
		},
		{
			name: "zero version poll interval",
			config: &Config{
				KafkaBrokers:        "localhost:9092",
				AlertsNewTopic:      "alerts.new",
				AlertsMatchedTopic:  "alerts.matched",
				RuleChangedTopic:    "rule.changed",
				ConsumerGroupID:     "evaluator-group",
				RuleChangedGroupID:  "evaluator-rule-changed-group",
				RedisAddr:           "localhost:6379",
				VersionPollInterval: 0,
			},
			wantErr: true,
			errMsg:  "version-poll-interval must be > 0",
		},
		{
			name: "negative version poll interval",
			config: &Config{
				KafkaBrokers:        "localhost:9092",
				AlertsNewTopic:      "alerts.new",
				AlertsMatchedTopic:  "alerts.matched",
				RuleChangedTopic:    "rule.changed",
				ConsumerGroupID:     "evaluator-group",
				RuleChangedGroupID:  "evaluator-rule-changed-group",
				RedisAddr:           "localhost:6379",
				VersionPollInterval: -1 * time.Second,
			},
			wantErr: true,
			errMsg:  "version-poll-interval must be > 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && err.Error() != tt.errMsg {
				t.Errorf("Config.Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}
