package producer

import (
	"testing"
)

func TestNewProducer(t *testing.T) {
	tests := []struct {
		name    string
		brokers string
		topic   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid producer",
			brokers: "localhost:9092",
			topic:   "notifications.ready",
			wantErr: false,
		},
		{
			name:    "empty brokers",
			brokers: "",
			topic:   "notifications.ready",
			wantErr: true,
			errMsg:  "brokers cannot be empty",
		},
		{
			name:    "empty topic",
			brokers: "localhost:9092",
			topic:   "",
			wantErr: true,
			errMsg:  "topic cannot be empty",
		},
		{
			name:    "multiple brokers",
			brokers: "localhost:9092,localhost:9093",
			topic:   "notifications.ready",
			wantErr: false,
		},
		{
			name:    "brokers with spaces",
			brokers: "localhost:9092, localhost:9093",
			topic:   "notifications.ready",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This will try to connect to Kafka, which may fail in test environment
			// In a real scenario, you'd use dependency injection or a factory pattern
			// For now, we test the validation logic
			producer, err := NewProducer(tt.brokers, tt.topic)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewProducer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if err.Error() != tt.errMsg {
					t.Errorf("NewProducer() error = %v, want error message %v", err.Error(), tt.errMsg)
				}
			}
			if !tt.wantErr && producer != nil {
				producer.Close()
			}
		})
	}
}

// Note: Publish and Close tests require a real Kafka instance
// or refactoring to use interfaces for dependency injection.
// The validation tests above cover the NewProducer function.
// For full coverage of Publish and Close, integration tests
// with a test Kafka instance are needed.
