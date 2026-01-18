package processor

import (
	"testing"

	"evaluator/internal/consumer"
	"evaluator/internal/indexes"
	"evaluator/internal/matcher"
	"evaluator/internal/producer"
	"evaluator/internal/snapshot"
)

func TestNewProcessor(t *testing.T) {
	// Test with real instances (will fail if Kafka not available, but that's OK)
	consumer, err := consumer.NewConsumer("localhost:9092", "alerts.new", "test-group")
	if err != nil {
		t.Skipf("Skipping test: Kafka not available: %v", err)
		return
	}
	defer consumer.Close()

	producer, err := producer.NewProducer("localhost:9092", "alerts.matched")
	if err != nil {
		t.Skipf("Skipping test: Kafka not available: %v", err)
		return
	}
	defer producer.Close()

	snap := &snapshot.Snapshot{
		BySeverity: map[string][]int{"HIGH": {1}},
		BySource:   map[string][]int{"service-a": {1}},
		ByName:     map[string][]int{"disk-full": {1}},
		Rules:      map[int]snapshot.RuleInfo{1: {RuleID: "rule-1", ClientID: "client-1"}},
	}
	idx := indexes.NewIndexes(snap)
	m := matcher.NewMatcher(idx)

	processor := NewProcessor(consumer, producer, m)
	if processor == nil {
		t.Fatal("NewProcessor() returned nil")
	}
}

// Note: ProcessAlerts() tests require real Kafka instances and are better suited for integration tests.
// The constructor test above validates that NewProcessor works correctly.
