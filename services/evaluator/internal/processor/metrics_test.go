package processor

import (
	"testing"
	"time"
)

func TestNoOpMetrics(t *testing.T) {
	// NoOpMetrics should be safe to call without panicking
	m := NoOpMetrics{}

	// All methods should be no-ops
	m.RecordReceived()
	m.RecordPublished()
	m.RecordError()
	m.RecordProcessed(100 * time.Millisecond)
	m.IncrementCustom("test")
	m.AddCustom("test", 42)
}

func TestWrapMetrics_Nil(t *testing.T) {
	// wrapMetrics(nil) should return NoOpMetrics
	m := wrapMetrics(nil)

	_, ok := m.(NoOpMetrics)
	if !ok {
		t.Errorf("wrapMetrics(nil) should return NoOpMetrics, got %T", m)
	}
}

// mockCollector implements metricsCollector for testing
type mockCollector struct {
	receivedCount  int
	publishedCount int
	errorCount     int
	processedCount int
	customCounts   map[string]uint64
}

func newMockCollector() *mockCollector {
	return &mockCollector{
		customCounts: make(map[string]uint64),
	}
}

func (m *mockCollector) RecordReceived()                   { m.receivedCount++ }
func (m *mockCollector) RecordPublished()                  { m.publishedCount++ }
func (m *mockCollector) RecordError()                      { m.errorCount++ }
func (m *mockCollector) RecordProcessed(time.Duration)     { m.processedCount++ }
func (m *mockCollector) IncrementCustom(name string)       { m.customCounts[name]++ }
func (m *mockCollector) AddCustom(name string, val uint64) { m.customCounts[name] += val }

func TestWrapMetrics_Collector(t *testing.T) {
	mock := newMockCollector()
	m := wrapMetrics(mock)

	// Verify it's wrapped in an adapter
	_, ok := m.(*collectorAdapter)
	if !ok {
		t.Errorf("wrapMetrics(collector) should return *collectorAdapter, got %T", m)
	}

	// Verify calls are forwarded
	m.RecordReceived()
	m.RecordReceived()
	m.RecordPublished()
	m.RecordError()
	m.RecordProcessed(time.Millisecond)
	m.IncrementCustom("alerts_matched")
	m.AddCustom("rules_count", 100)

	if mock.receivedCount != 2 {
		t.Errorf("receivedCount = %d, want 2", mock.receivedCount)
	}
	if mock.publishedCount != 1 {
		t.Errorf("publishedCount = %d, want 1", mock.publishedCount)
	}
	if mock.errorCount != 1 {
		t.Errorf("errorCount = %d, want 1", mock.errorCount)
	}
	if mock.processedCount != 1 {
		t.Errorf("processedCount = %d, want 1", mock.processedCount)
	}
	if mock.customCounts["alerts_matched"] != 1 {
		t.Errorf("customCounts[alerts_matched] = %d, want 1", mock.customCounts["alerts_matched"])
	}
	if mock.customCounts["rules_count"] != 100 {
		t.Errorf("customCounts[rules_count] = %d, want 100", mock.customCounts["rules_count"])
	}
}
