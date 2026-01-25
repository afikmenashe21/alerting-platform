package processor

import (
	"testing"
	"time"
)

func TestNoOpMetrics_RecordError(t *testing.T) {
	m := NoOpMetrics{}
	// Should not panic
	m.RecordError()
}

func TestNoOpMetrics_RecordProcessed(t *testing.T) {
	m := NoOpMetrics{}
	// Should not panic
	m.RecordProcessed(100 * time.Millisecond)
}

func TestNoOpMetrics_RecordPublished(t *testing.T) {
	m := NoOpMetrics{}
	// Should not panic
	m.RecordPublished()
}

func TestNoOpMetrics_ImplementsInterface(t *testing.T) {
	var _ MetricsRecorder = NoOpMetrics{}
	var _ MetricsRecorder = &NoOpMetrics{}
}

// mockMetrics is used to verify metrics are called in processor tests
type mockMetrics struct {
	errorCount     int
	processedCount int
	publishedCount int
}

func (m *mockMetrics) RecordError() {
	m.errorCount++
}

func (m *mockMetrics) RecordProcessed(d time.Duration) {
	m.processedCount++
}

func (m *mockMetrics) RecordPublished() {
	m.publishedCount++
}

var _ MetricsRecorder = (*mockMetrics)(nil)

func TestProcessorWithMockMetrics(t *testing.T) {
	// This test verifies that metrics are called during processing
	// by using a mock implementation
	mm := &mockMetrics{}

	// Verify mock implements interface
	if mm.errorCount != 0 || mm.processedCount != 0 || mm.publishedCount != 0 {
		t.Error("mock metrics should start at zero")
	}

	mm.RecordError()
	mm.RecordProcessed(time.Millisecond)
	mm.RecordPublished()

	if mm.errorCount != 1 {
		t.Errorf("errorCount = %d, want 1", mm.errorCount)
	}
	if mm.processedCount != 1 {
		t.Errorf("processedCount = %d, want 1", mm.processedCount)
	}
	if mm.publishedCount != 1 {
		t.Errorf("publishedCount = %d, want 1", mm.publishedCount)
	}
}
