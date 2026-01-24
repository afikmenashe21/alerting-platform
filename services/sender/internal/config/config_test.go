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
				NotificationsReadyTopic: "notifications.ready",
				ConsumerGroupID:         "sender-group",
				PostgresDSN:             "postgres://user:pass@localhost:5432/db",
				RedisAddr:               "localhost:6379",
			},
			wantErr: false,
		},
		{
			name: "empty kafka brokers",
			config: Config{
				KafkaBrokers:            "",
				NotificationsReadyTopic: "notifications.ready",
				ConsumerGroupID:         "sender-group",
				PostgresDSN:             "postgres://user:pass@localhost:5432/db",
				RedisAddr:               "localhost:6379",
			},
			wantErr: true,
			errMsg:  "kafka-brokers cannot be empty",
		},
		{
			name: "empty notifications ready topic",
			config: Config{
				KafkaBrokers:            "localhost:9092",
				NotificationsReadyTopic: "",
				ConsumerGroupID:         "sender-group",
				PostgresDSN:             "postgres://user:pass@localhost:5432/db",
				RedisAddr:               "localhost:6379",
			},
			wantErr: true,
			errMsg:  "notifications-ready-topic cannot be empty",
		},
		{
			name: "empty consumer group id",
			config: Config{
				KafkaBrokers:            "localhost:9092",
				NotificationsReadyTopic: "notifications.ready",
				ConsumerGroupID:         "",
				PostgresDSN:             "postgres://user:pass@localhost:5432/db",
				RedisAddr:               "localhost:6379",
			},
			wantErr: true,
			errMsg:  "consumer-group-id cannot be empty",
		},
		{
			name: "empty postgres dsn",
			config: Config{
				KafkaBrokers:            "localhost:9092",
				NotificationsReadyTopic: "notifications.ready",
				ConsumerGroupID:         "sender-group",
				PostgresDSN:             "",
				RedisAddr:               "localhost:6379",
			},
			wantErr: true,
			errMsg:  "postgres-dsn cannot be empty",
		},
		{
			name: "empty redis addr",
			config: Config{
				KafkaBrokers:            "localhost:9092",
				NotificationsReadyTopic: "notifications.ready",
				ConsumerGroupID:         "sender-group",
				PostgresDSN:             "postgres://user:pass@localhost:5432/db",
				RedisAddr:               "",
			},
			wantErr: true,
			errMsg:  "redis-addr cannot be empty",
		},
		{
			name: "all fields empty",
			config: Config{
				KafkaBrokers:            "",
				NotificationsReadyTopic: "",
				ConsumerGroupID:         "",
				PostgresDSN:             "",
				RedisAddr:               "",
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
