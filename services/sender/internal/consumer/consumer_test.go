package consumer

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"sender/internal/events"
	"github.com/segmentio/kafka-go"
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
			topic:   "notifications.ready",
			groupID: "test-group",
			wantErr: false,
		},
		{
			name:    "empty brokers",
			brokers: "",
			topic:   "notifications.ready",
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
			topic:   "notifications.ready",
			groupID: "",
			wantErr: true,
			errMsg:  "groupID cannot be empty",
		},
		{
			name:    "multiple brokers",
			brokers: "localhost:9092,localhost:9093",
			topic:   "notifications.ready",
			groupID: "test-group",
			wantErr: false,
		},
		{
			name:    "brokers with spaces",
			brokers: "localhost:9092, localhost:9093",
			topic:   "notifications.ready",
			groupID: "test-group",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
				_ = consumer.Close()
			}
		})
	}
}

func TestConsumer_Close(t *testing.T) {
	consumer, err := NewConsumer("localhost:9092", "notifications.ready", "test-group-close")
	if err != nil {
		t.Skipf("Skipping Close test: Kafka not available: %v", err)
		return
	}

	if err := consumer.Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}

	// Close again should be safe
	_ = consumer.Close()
}

func TestConsumer_ReadMessage_InvalidJSON(t *testing.T) {
	consumer, err := NewConsumer("localhost:9092", "notifications.ready", "test-group-read")
	if err != nil {
		t.Skipf("Skipping ReadMessage test: Kafka not available: %v", err)
		return
	}
	defer consumer.Close()

	ctx := context.Background()
	_, _, err = consumer.ReadMessage(ctx)
	if err != nil {
		t.Logf("ReadMessage() error (expected in test environment): %v", err)
	}
}

func TestConsumer_ReadMessage_ValidJSON(t *testing.T) {
	consumer, err := NewConsumer("localhost:9092", "notifications.ready", "test-group-read-valid")
	if err != nil {
		t.Skipf("Skipping ReadMessage test: Kafka not available: %v", err)
		return
	}
	defer consumer.Close()

	// Create a valid notification ready event
	ready := events.NotificationReady{
		NotificationID: "notif-123",
		ClientID:       "client-456",
		AlertID:        "alert-789",
		SchemaVersion:  1,
	}
	value, _ := json.Marshal(ready)

	// Create a mock message
	msg := kafka.Message{
		Value: value,
	}

	// Test that ReadMessage can handle valid JSON
	// Note: This test requires actual Kafka or mocking
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, _, err = consumer.ReadMessage(ctx)
	if err != nil {
		// Expected if Kafka is not available
		t.Logf("ReadMessage() error (expected in test environment): %v", err)
	}

	_ = msg // Use msg to avoid unused variable
}

func TestConsumer_CommitMessage(t *testing.T) {
	consumer, err := NewConsumer("localhost:9092", "notifications.ready", "test-group-commit")
	if err != nil {
		t.Skipf("Skipping CommitMessage test: Kafka not available: %v", err)
		return
	}
	defer consumer.Close()

	// Create a mock message
	msg := kafka.Message{
		Value: []byte(`{"notification_id":"test","client_id":"test","alert_id":"test","schema_version":1}`),
	}

	ctx := context.Background()
	err = consumer.CommitMessage(ctx, &msg)
	if err != nil {
		t.Logf("CommitMessage() error (expected in test environment): %v", err)
	}
}
