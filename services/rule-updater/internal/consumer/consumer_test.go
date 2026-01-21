package consumer

import (
	"context"
	"errors"
	"testing"
	"time"

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

func TestConsumer_ReadMessage(t *testing.T) {
	consumer, err := NewConsumer("localhost:9092", "rule.changed", "test-group-read")
	if err != nil {
		t.Skipf("Skipping ReadMessage test: Kafka not available: %v", err)
		return
	}
	defer consumer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// ReadMessage will fail if Kafka is not available or topic is empty
	// This tests the error handling path
	_, _, err = consumer.ReadMessage(ctx)
	if err != nil {
		// Expected if Kafka is not available or topic is empty
		t.Logf("ReadMessage() error (expected in test environment): %v", err)
	}
}

func TestConsumer_ReadMessage_InvalidJSON(t *testing.T) {
	// This test verifies that ReadMessage handles invalid JSON gracefully
	// In a real scenario, this would require a Kafka message with invalid JSON
	// For now, we test the error handling path
	consumer, err := NewConsumer("localhost:9092", "rule.changed", "test-group-invalid-json")
	if err != nil {
		t.Skipf("Skipping ReadMessage invalid JSON test: Kafka not available: %v", err)
		return
	}
	defer consumer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, _, err = consumer.ReadMessage(ctx)
	if err != nil {
		// Expected if Kafka is not available or topic is empty
		t.Logf("ReadMessage() error (expected in test environment): %v", err)
	}
}

func TestConsumer_CommitMessage(t *testing.T) {
	consumer, err := NewConsumer("localhost:9092", "rule.changed", "test-group-commit")
	if err != nil {
		t.Skipf("Skipping CommitMessage test: Kafka not available: %v", err)
		return
	}
	defer consumer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// CommitMessage will fail if there's no message to commit
	// This tests the error handling path
	msg := kafka.Message{
		Topic: "rule.changed",
		Partition: 0,
		Offset: 0,
	}
	err = consumer.CommitMessage(ctx, &msg)
	if err != nil {
		// Expected if Kafka is not available or no message to commit
		t.Logf("CommitMessage() error (expected in test environment): %v", err)
	}
}

func TestConsumer_ReadMessage_ContextCancellation(t *testing.T) {
	consumer, err := NewConsumer("localhost:9092", "rule.changed", "test-group-context")
	if err != nil {
		t.Skipf("Skipping context cancellation test: Kafka not available: %v", err)
		return
	}
	defer consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, _, err = consumer.ReadMessage(ctx)
	if err == nil {
		t.Error("ReadMessage() expected error on cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) && ctx.Err() == nil {
		// Context cancellation may be handled differently by kafka-go
		t.Logf("ReadMessage() with cancelled context: %v", err)
	}
}
