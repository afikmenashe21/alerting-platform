package processor

import (
	"context"
	"errors"
	"testing"

	"aggregator/internal/events"
)

func TestNewProcessor(t *testing.T) {
	// Create processor with nil values to test initialization
	proc := NewProcessor(nil, nil, nil)

	if proc == nil {
		t.Error("NewProcessor() returned nil")
	}
	if proc.reader != nil {
		t.Error("NewProcessor() reader should be nil")
	}
	if proc.publisher != nil {
		t.Error("NewProcessor() publisher should be nil")
	}
	if proc.storage != nil {
		t.Error("NewProcessor() storage should be nil")
	}
	// Check that metrics is not nil (NoOpMetrics is used by default)
	if proc.metrics == nil {
		t.Error("NewProcessor() metrics should not be nil (should be NoOpMetrics)")
	}
}

func TestNewProcessorWithMetrics(t *testing.T) {
	t.Run("with nil metrics uses NoOpMetrics", func(t *testing.T) {
		proc := NewProcessorWithMetrics(nil, nil, nil, nil)
		if proc.metrics == nil {
			t.Error("NewProcessorWithMetrics() with nil should use NoOpMetrics")
		}
	})

	t.Run("with custom metrics uses provided metrics", func(t *testing.T) {
		customMetrics := &NoOpMetrics{}
		proc := NewProcessorWithMetrics(nil, nil, nil, customMetrics)
		if proc.metrics != customMetrics {
			t.Error("NewProcessorWithMetrics() should use provided metrics")
		}
	})
}

func TestNoOpMetrics(t *testing.T) {
	// Ensure NoOpMetrics implements MetricsRecorder and doesn't panic
	m := &NoOpMetrics{}

	// These should all be no-ops and not panic
	m.RecordReceived()
	m.RecordProcessed(0)
	m.RecordPublished()
	m.RecordError()
	m.IncrementCustom("test")
}

func TestProcessMessage_NewNotification(t *testing.T) {
	// Setup
	notificationID := "notif-123"
	storage := &FakeStorage{InsertResult: &notificationID}
	publisher := &FakePublisher{}
	metrics := NewFakeMetrics()

	proc := NewProcessorWithMetrics(nil, publisher, storage, metrics)

	matched := &events.AlertMatched{
		AlertID:  "alert-1",
		ClientID: "client-1",
		Severity: "HIGH",
		Source:   "payments",
		Name:     "transaction_failed",
		Context:  map[string]string{"key": "value"},
		RuleIDs:  []string{"rule-1", "rule-2"},
	}

	// Execute
	result := proc.processMessage(context.Background(), matched)

	// Verify
	if !result {
		t.Error("processMessage() should return true for successful processing")
	}

	// Check storage was called
	if len(storage.InsertedNotifications) != 1 {
		t.Fatalf("Expected 1 insert call, got %d", len(storage.InsertedNotifications))
	}
	insert := storage.InsertedNotifications[0]
	if insert.ClientID != "client-1" {
		t.Errorf("Expected ClientID 'client-1', got '%s'", insert.ClientID)
	}
	if insert.AlertID != "alert-1" {
		t.Errorf("Expected AlertID 'alert-1', got '%s'", insert.AlertID)
	}

	// Check publisher was called
	if len(publisher.Published) != 1 {
		t.Fatalf("Expected 1 publish call, got %d", len(publisher.Published))
	}
	published := publisher.Published[0]
	if published.NotificationID != notificationID {
		t.Errorf("Expected NotificationID '%s', got '%s'", notificationID, published.NotificationID)
	}

	// Check metrics
	if metrics.ProcessedCount != 1 {
		t.Errorf("Expected ProcessedCount 1, got %d", metrics.ProcessedCount)
	}
	if metrics.PublishedCount != 1 {
		t.Errorf("Expected PublishedCount 1, got %d", metrics.PublishedCount)
	}
	if metrics.CustomIncrements["notifications_created"] != 1 {
		t.Errorf("Expected notifications_created 1, got %d", metrics.CustomIncrements["notifications_created"])
	}
	if metrics.ErrorCount != 0 {
		t.Errorf("Expected ErrorCount 0, got %d", metrics.ErrorCount)
	}
}

func TestProcessMessage_DuplicateNotification(t *testing.T) {
	// Setup - InsertResult is nil, meaning the notification already exists
	storage := &FakeStorage{InsertResult: nil}
	publisher := &FakePublisher{}
	metrics := NewFakeMetrics()

	proc := NewProcessorWithMetrics(nil, publisher, storage, metrics)

	matched := &events.AlertMatched{
		AlertID:  "alert-1",
		ClientID: "client-1",
		Severity: "HIGH",
	}

	// Execute
	result := proc.processMessage(context.Background(), matched)

	// Verify
	if !result {
		t.Error("processMessage() should return true for duplicate (no error)")
	}

	// Check publisher was NOT called
	if len(publisher.Published) != 0 {
		t.Errorf("Expected 0 publish calls for duplicate, got %d", len(publisher.Published))
	}

	// Check metrics
	if metrics.CustomIncrements["notifications_deduplicated"] != 1 {
		t.Errorf("Expected notifications_deduplicated 1, got %d", metrics.CustomIncrements["notifications_deduplicated"])
	}
	if metrics.CustomIncrements["notifications_created"] != 0 {
		t.Errorf("Expected notifications_created 0, got %d", metrics.CustomIncrements["notifications_created"])
	}
}

func TestProcessMessage_StorageError(t *testing.T) {
	// Setup
	storage := &FakeStorage{InsertErr: errors.New("database connection failed")}
	publisher := &FakePublisher{}
	metrics := NewFakeMetrics()

	proc := NewProcessorWithMetrics(nil, publisher, storage, metrics)

	matched := &events.AlertMatched{
		AlertID:  "alert-1",
		ClientID: "client-1",
	}

	// Execute
	result := proc.processMessage(context.Background(), matched)

	// Verify
	if result {
		t.Error("processMessage() should return false on storage error")
	}

	// Check publisher was NOT called
	if len(publisher.Published) != 0 {
		t.Errorf("Expected 0 publish calls on error, got %d", len(publisher.Published))
	}

	// Check metrics
	if metrics.ErrorCount != 1 {
		t.Errorf("Expected ErrorCount 1, got %d", metrics.ErrorCount)
	}
	if metrics.ProcessedCount != 0 {
		t.Errorf("Expected ProcessedCount 0, got %d", metrics.ProcessedCount)
	}
}

func TestProcessMessage_PublishError(t *testing.T) {
	// Setup
	notificationID := "notif-123"
	storage := &FakeStorage{InsertResult: &notificationID}
	publisher := &FakePublisher{PublishErr: errors.New("kafka connection failed")}
	metrics := NewFakeMetrics()

	proc := NewProcessorWithMetrics(nil, publisher, storage, metrics)

	matched := &events.AlertMatched{
		AlertID:  "alert-1",
		ClientID: "client-1",
	}

	// Execute
	result := proc.processMessage(context.Background(), matched)

	// Verify
	if result {
		t.Error("processMessage() should return false on publish error")
	}

	// Check metrics
	if metrics.ErrorCount != 1 {
		t.Errorf("Expected ErrorCount 1, got %d", metrics.ErrorCount)
	}
	if metrics.ProcessedCount != 0 {
		t.Errorf("Expected ProcessedCount 0 (not processed due to error), got %d", metrics.ProcessedCount)
	}
}

func TestPublishNotification(t *testing.T) {
	// Setup
	publisher := &FakePublisher{}
	metrics := NewFakeMetrics()

	proc := NewProcessorWithMetrics(nil, publisher, nil, metrics)

	matched := &events.AlertMatched{
		AlertID:       "alert-1",
		ClientID:      "client-1",
		SchemaVersion: 1,
		RuleIDs:       []string{"rule-1"},
	}

	// Execute
	result := proc.publishNotification(context.Background(), matched, "notif-123")

	// Verify
	if !result {
		t.Error("publishNotification() should return true on success")
	}

	if len(publisher.Published) != 1 {
		t.Fatalf("Expected 1 publish call, got %d", len(publisher.Published))
	}

	published := publisher.Published[0]
	if published.NotificationID != "notif-123" {
		t.Errorf("Expected NotificationID 'notif-123', got '%s'", published.NotificationID)
	}
	if published.AlertID != "alert-1" {
		t.Errorf("Expected AlertID 'alert-1', got '%s'", published.AlertID)
	}
	if published.ClientID != "client-1" {
		t.Errorf("Expected ClientID 'client-1', got '%s'", published.ClientID)
	}

	if metrics.PublishedCount != 1 {
		t.Errorf("Expected PublishedCount 1, got %d", metrics.PublishedCount)
	}
}
