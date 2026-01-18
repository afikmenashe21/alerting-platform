package processor

import (
	"context"
	"testing"
	"time"

	"evaluator/internal/indexes"
	"evaluator/internal/matcher"
	"evaluator/internal/reloader"
	"evaluator/internal/ruleconsumer"
	"evaluator/internal/snapshot"

	"github.com/redis/go-redis/v9"
)

func TestNewRuleHandler(t *testing.T) {
	// Test with real instances (will fail if Kafka/Redis not available, but that's OK)
	consumer, err := ruleconsumer.NewConsumer("localhost:9092", "rule.changed", "test-group")
	if err != nil {
		t.Skipf("Skipping test: Kafka not available: %v", err)
		return
	}
	defer consumer.Close()

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Skipping test: Redis not available: %v", err)
		return
	}

	loader := snapshot.NewLoader(client)
	snap := &snapshot.Snapshot{
		BySeverity: map[string][]int{"HIGH": {1}},
		BySource:   map[string][]int{"service-a": {1}},
		ByName:     map[string][]int{"disk-full": {1}},
		Rules:      map[int]snapshot.RuleInfo{1: {RuleID: "rule-1", ClientID: "client-1"}},
	}
	idx := indexes.NewIndexes(snap)
	m := matcher.NewMatcher(idx)
	reload := reloader.NewReloader(loader, m, 5*time.Second)

	handler := NewRuleHandler(consumer, reload)
	if handler == nil {
		t.Fatal("NewRuleHandler() returned nil")
	}
}

// Note: HandleRuleChanged() tests require real Kafka/Redis instances and are better suited for integration tests.
// The constructor test above validates that NewRuleHandler works correctly.
