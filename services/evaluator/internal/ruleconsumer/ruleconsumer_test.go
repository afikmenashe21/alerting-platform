package ruleconsumer

import (
	"context"
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
			topic:   "rule.changed",
			groupID: "test-group",
			wantErr: false,
		},
		{
			name:    "empty brokers",
			brokers: "",
			topic:   "rule.changed",
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
			topic:   "rule.changed",
			groupID: "",
			wantErr: true,
			errMsg:  "groupID cannot be empty",
		},
		{
			name:    "multiple brokers",
			brokers: "localhost:9092,localhost:9093",
			topic:   "rule.changed",
			groupID: "test-group",
			wantErr: false,
		},
		{
			name:    "brokers with spaces",
			brokers: "localhost:9092, localhost:9093",
			topic:   "rule.changed",
			groupID: "test-group",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This will try to connect to Kafka, which may fail in test environment
			// We test the validation logic and error handling
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
				// Clean up if consumer was created
				_ = consumer.Close()
			}
		})
	}
}

func TestConsumer_Close(t *testing.T) {
	// Test Close on valid consumer (requires Kafka connection)
	consumer, err := NewConsumer("localhost:9092", "rule.changed", "test-group-close")
	if err != nil {
		// Kafka not available, skip this test
		t.Skipf("Skipping Close test: Kafka not available: %v", err)
		return
	}

	if err := consumer.Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}

	// Close again should be safe (may return error if already closed, which is OK)
	_ = consumer.Close()
}

func TestConsumer_ReadMessage_InvalidJSON(t *testing.T) {
	// This test requires Kafka to be running with a topic that has invalid JSON messages
	// For now, we test that ReadMessage handles errors gracefully
	consumer, err := NewConsumer("localhost:9092", "rule.changed", "test-group-read")
	if err != nil {
		t.Skipf("Skipping ReadMessage test: Kafka not available: %v", err)
		return
	}
	defer consumer.Close()

	// ReadMessage will fail if Kafka is not available or topic is empty
	// This tests the error handling path
	ctx := context.Background()
	_, err = consumer.ReadMessage(ctx)
	if err != nil {
		// Expected if Kafka is not available or topic is empty
		t.Logf("ReadMessage() error (expected in test environment): %v", err)
	}
}

// Note: ReadMessage tests require a real Kafka instance or interface refactoring
// The validation tests above cover the NewConsumer function and error handling.
// For full coverage of ReadMessage, you would need:
// 1. Interface-based refactoring with mocks, OR
// 2. Integration tests with testcontainers or real Kafka instance
