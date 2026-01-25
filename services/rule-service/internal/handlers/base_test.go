// Package handlers provides tests for base handler functionality.
package handlers

import (
	"testing"

	"rule-service/internal/database"
	"rule-service/internal/producer"
)

// TestNewHandlers tests the NewHandlers constructor.
func TestNewHandlers(t *testing.T) {
	// Create mock dependencies
	db := &database.DB{}         // Note: This is a nil pointer, but we're just testing the constructor
	prod := &producer.Producer{} // Note: This is a nil pointer, but we're just testing the constructor

	// Test constructor (metricsCollector can be nil - will use NoOpMetrics)
	h := NewHandlers(db, prod, nil)
	if h == nil {
		t.Fatal("NewHandlers() returned nil")
	}
	if h.db != db {
		t.Errorf("NewHandlers() db = %v, want %v", h.db, db)
	}
	if h.producer != prod {
		t.Errorf("NewHandlers() producer = %v, want %v", h.producer, prod)
	}
	// Verify metrics is not nil (should be NoOpMetrics)
	if h.metrics == nil {
		t.Error("NewHandlers() metrics should not be nil (should be NoOpMetrics)")
	}
}

// TestNewHandlersWithDeps tests the interface-based constructor.
func TestNewHandlersWithDeps(t *testing.T) {
	mockDB := &mockRepository{}
	mockPub := &mockPublisher{}
	mockMetrics := &mockMetrics{}

	h := NewHandlersWithDeps(mockDB, mockPub, mockMetrics)

	if h == nil {
		t.Fatal("NewHandlersWithDeps() returned nil")
	}
	if h.db != mockDB {
		t.Error("NewHandlersWithDeps() did not set db correctly")
	}
	if h.producer != mockPub {
		t.Error("NewHandlersWithDeps() did not set producer correctly")
	}
	if h.metrics != mockMetrics {
		t.Error("NewHandlersWithDeps() did not set metrics correctly")
	}
}

// TestNewHandlersWithDeps_NilMetrics tests that nil metrics defaults to NoOpMetrics.
func TestNewHandlersWithDeps_NilMetrics(t *testing.T) {
	h := NewHandlersWithDeps(&mockRepository{}, &mockPublisher{}, nil)

	if h.metrics == nil {
		t.Error("metrics should not be nil when passed nil")
	}
	// Should be NoOpMetrics
	if _, ok := h.metrics.(NoOpMetrics); !ok {
		t.Error("metrics should be NoOpMetrics when passed nil")
	}
}

// TestNoOpMetrics ensures NoOpMetrics methods don't panic.
func TestNoOpMetrics(t *testing.T) {
	m := NoOpMetrics{}

	// All methods should complete without panic
	m.RecordReceived()
	m.RecordProcessed(0)
	m.RecordPublished()
	m.RecordError()
	m.IncrementCustom("test")
}
