// Package producer provides tests for Kafka producer functionality.
package producer

import (
	"context"
	"strings"
	"testing"
	"time"

	"rule-service/internal/events"
)

// TestNewProducer tests the NewProducer constructor with various scenarios.
func TestNewProducer(t *testing.T) {
	tests := []struct {
		name    string
		brokers string
		topic   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty brokers",
			brokers: "",
			topic:   "rule.changed",
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
			name:    "valid config",
			brokers: "localhost:9092",
			topic:   "rule.changed",
			wantErr: false,
		},
		{
			name:    "multiple brokers",
			brokers: "localhost:9092,localhost:9093",
			topic:   "rule.changed",
			wantErr: false,
		},
		{
			name:    "brokers with spaces",
			brokers: "localhost:9092, localhost:9093",
			topic:   "rule.changed",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This will try to connect to Kafka, which may fail in test environment
			// We test the validation logic and error handling
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
				// Clean up if producer was created
				_ = producer.Close()
			}
		})
	}
}

// TestProducer_Close tests the Close method.
func TestProducer_Close(t *testing.T) {
	producer, err := NewProducer("localhost:9092", "rule.changed")
	if err != nil {
		// Kafka not available, skip this test
		t.Skipf("Skipping Close test: Kafka not available: %v", err)
		return
	}

	if err := producer.Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}

	// Close again should be safe (may return error if already closed, which is OK)
	_ = producer.Close()
}

// TestProducer_Publish tests the Publish method.
func TestProducer_Publish(t *testing.T) {
	producer, err := NewProducer("localhost:9092", "rule.changed")
	if err != nil {
		// Kafka not available, skip this test
		t.Skipf("Skipping Publish test: Kafka not available: %v", err)
		return
	}
	defer producer.Close()

	ctx := context.Background()

	tests := []struct {
		name    string
		event   *events.RuleChanged
		wantErr bool
	}{
		{
			name: "CREATED action",
			event: &events.RuleChanged{
				RuleID:        "rule-1",
				ClientID:       "client-1",
				Action:         events.ActionCreated,
				Version:        1,
				UpdatedAt:      time.Now().Unix(),
				SchemaVersion:  1,
			},
			wantErr: false,
		},
		{
			name: "UPDATED action",
			event: &events.RuleChanged{
				RuleID:        "rule-1",
				ClientID:       "client-1",
				Action:         events.ActionUpdated,
				Version:        2,
				UpdatedAt:      time.Now().Unix(),
				SchemaVersion:  1,
			},
			wantErr: false,
		},
		{
			name: "DELETED action",
			event: &events.RuleChanged{
				RuleID:        "rule-1",
				ClientID:       "client-1",
				Action:         events.ActionDeleted,
				Version:        5,
				UpdatedAt:      time.Now().Unix(),
				SchemaVersion:  1,
			},
			wantErr: false,
		},
		{
			name: "DISABLED action",
			event: &events.RuleChanged{
				RuleID:        "rule-1",
				ClientID:       "client-1",
				Action:         events.ActionDisabled,
				Version:        3,
				UpdatedAt:      time.Now().Unix(),
				SchemaVersion:  1,
			},
			wantErr: false,
		},
	}

	// Test first event to check if Kafka is available
	firstErr := producer.Publish(ctx, tests[0].event)
	if firstErr != nil && strings.Contains(firstErr.Error(), "connection refused") {
		t.Skipf("Skipping Publish tests: Kafka not available: %v", firstErr)
		return
	}

	// If Kafka is available, run all tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := producer.Publish(ctx, tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("Publish() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestProducer_Publish_ContextCancellation tests Publish with cancelled context.
func TestProducer_Publish_ContextCancellation(t *testing.T) {
	producer, err := NewProducer("localhost:9092", "rule.changed")
	if err != nil {
		t.Skipf("Skipping context cancellation test: Kafka not available: %v", err)
		return
	}
	defer producer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	event := &events.RuleChanged{
		RuleID:        "rule-1",
		ClientID:       "client-1",
		Action:         events.ActionCreated,
		Version:        1,
		UpdatedAt:      time.Now().Unix(),
		SchemaVersion:  1,
	}

	// Publish should fail with context cancelled
	err = producer.Publish(ctx, event)
	if err == nil {
		t.Error("Publish() expected error with cancelled context")
	}
}

// TestCreateTopicIfNotExists tests the createTopicIfNotExists function indirectly.
// This is tested through NewProducer which calls it.
func TestCreateTopicIfNotExists(t *testing.T) {
	// This is tested indirectly through NewProducer
	// The function logs warnings but doesn't fail producer creation
	producer, err := NewProducer("localhost:9092", "test-topic-creation")
	if err != nil {
		// Kafka not available, skip
		t.Skipf("Skipping topic creation test: Kafka not available: %v", err)
		return
	}
	defer producer.Close()

	// If we got here, the producer was created successfully
	// The createTopicIfNotExists function was called and handled gracefully
}
