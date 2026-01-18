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
				KafkaBrokers:            "localhost:9092",
				AlertsMatchedTopic:      "alerts.matched",
				NotificationsReadyTopic: "notifications.ready",
				ConsumerGroupID:         "aggregator-group",
				PostgresDSN:             "postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable",
			},
			wantErr: false,
		},
		{
			name: "missing kafka-brokers",
			config: Config{
				AlertsMatchedTopic:      "alerts.matched",
				NotificationsReadyTopic: "notifications.ready",
				ConsumerGroupID:         "aggregator-group",
				PostgresDSN:             "postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable",
			},
			wantErr: true,
			errMsg:  "kafka-brokers cannot be empty",
		},
		{
			name: "missing alerts-matched-topic",
			config: Config{
				KafkaBrokers:            "localhost:9092",
				NotificationsReadyTopic: "notifications.ready",
				ConsumerGroupID:         "aggregator-group",
				PostgresDSN:             "postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable",
			},
			wantErr: true,
			errMsg:  "alerts-matched-topic cannot be empty",
		},
		{
			name: "missing notifications-ready-topic",
			config: Config{
				KafkaBrokers:       "localhost:9092",
				AlertsMatchedTopic: "alerts.matched",
				ConsumerGroupID:    "aggregator-group",
				PostgresDSN:         "postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable",
			},
			wantErr: true,
			errMsg:  "notifications-ready-topic cannot be empty",
		},
		{
			name: "missing consumer-group-id",
			config: Config{
				KafkaBrokers:            "localhost:9092",
				AlertsMatchedTopic:      "alerts.matched",
				NotificationsReadyTopic: "notifications.ready",
				PostgresDSN:             "postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable",
			},
			wantErr: true,
			errMsg:  "consumer-group-id cannot be empty",
		},
		{
			name: "missing postgres-dsn",
			config: Config{
				KafkaBrokers:            "localhost:9092",
				AlertsMatchedTopic:      "alerts.matched",
				NotificationsReadyTopic: "notifications.ready",
				ConsumerGroupID:         "aggregator-group",
			},
			wantErr: true,
			errMsg:  "postgres-dsn cannot be empty",
		},
		{
			name: "all fields empty",
			config: Config{
				KafkaBrokers:            "",
				AlertsMatchedTopic:      "",
				NotificationsReadyTopic: "",
				ConsumerGroupID:         "",
				PostgresDSN:             "",
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
