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

	// Test constructor (metricsReader and metricsCollector can be nil)
	h := NewHandlers(db, prod, nil, nil)
	if h == nil {
		t.Fatal("NewHandlers() returned nil")
	}
	if h.db != db {
		t.Errorf("NewHandlers() db = %v, want %v", h.db, db)
	}
	if h.producer != prod {
		t.Errorf("NewHandlers() producer = %v, want %v", h.producer, prod)
	}
}
