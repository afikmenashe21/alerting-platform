package consumer

import (
	"testing"
)

func TestNewConsumer(t *testing.T) {
	tests := []struct {
		name    string
		brokers string
		topic   string
		groupID string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid consumer",
			brokers: "localhost:9092",
			topic:   "alerts.matched",
			groupID: "test-group",
			wantErr: false,
		},
		{
			name:    "empty brokers",
			brokers: "",
			topic:   "alerts.matched",
			groupID: "test-group",
			wantErr: true,
			errMsg:  "brokers cannot be empty",
		},
		{
			name:    "empty topic",
			brokers: "localhost:9092",
			topic:   "",
			groupID: "test-group",
			wantErr: true,
			errMsg:  "topic cannot be empty",
		},
		{
			name:    "empty groupID",
			brokers: "localhost:9092",
			topic:   "alerts.matched",
			groupID: "",
			wantErr: true,
			errMsg:  "groupID cannot be empty",
		},
		{
			name:    "multiple brokers",
			brokers: "localhost:9092,localhost:9093",
			topic:   "alerts.matched",
			groupID: "test-group",
			wantErr: false,
		},
		{
			name:    "brokers with spaces",
			brokers: "localhost:9092, localhost:9093",
			topic:   "alerts.matched",
			groupID: "test-group",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This will try to connect to Kafka, which may fail in test environment
			// In a real scenario, you'd use dependency injection or a factory pattern
			// For now, we test the validation logic
			consumer, err := NewConsumer(tt.brokers, tt.topic, tt.groupID)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewConsumer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if err.Error() != tt.errMsg {
					t.Errorf("NewConsumer() error = %v, want error message %v", err.Error(), tt.errMsg)
				}
			}
			if !tt.wantErr && consumer != nil {
				consumer.Close()
			}
		})
	}
}

// Note: ReadMessage, CommitMessage, and Close tests require a real Kafka instance
// or refactoring to use interfaces for dependency injection.
// The validation tests above cover the NewConsumer function.
// For full coverage of ReadMessage, CommitMessage, and Close, integration tests
// with a test Kafka instance are needed.
