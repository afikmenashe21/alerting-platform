package processor

import (
	"testing"

	"aggregator/internal/consumer"
	"aggregator/internal/database"
	"aggregator/internal/producer"
)

func TestNewProcessor(t *testing.T) {
	// Create processor with nil values to test initialization
	// In real usage, these would be actual consumer, producer, and db instances
	var cons *consumer.Consumer
	var prod *producer.Producer
	var db *database.DB

	proc := NewProcessor(cons, prod, db)

	if proc == nil {
		t.Error("NewProcessor() returned nil")
	}
	if proc.consumer != cons {
		t.Error("NewProcessor() consumer not set correctly")
	}
	if proc.producer != prod {
		t.Error("NewProcessor() producer not set correctly")
	}
	if proc.db != db {
		t.Error("NewProcessor() db not set correctly")
	}
}

// Note: ProcessNotifications requires real Kafka and DB instances for full testing
// To achieve 100% coverage, you would need either:
// 1. Integration tests with testcontainers or real test infrastructure
// 2. Refactoring to use interfaces for dependency injection
// 
// The current implementation uses concrete types which makes unit testing difficult.
// For now, we test the constructor which is the only easily testable part.
