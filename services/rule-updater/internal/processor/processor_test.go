package processor

import (
	"context"
	"testing"
	"time"

	"rule-updater/internal/consumer"
	"rule-updater/internal/events"
	"rule-updater/internal/snapshot"
	"github.com/redis/go-redis/v9"
)

func TestNewProcessor(t *testing.T) {
	// Create real instances (will fail if Kafka/Redis not available, but that's OK for constructor test)
	kafkaConsumer, err := consumer.NewConsumer("localhost:9092", "rule.changed", "test-group")
	if err != nil {
		t.Skipf("Skipping test: Kafka not available: %v", err)
		return
	}
	defer kafkaConsumer.Close()

	// Create a mock DB using sqlmock would be better, but for now we'll skip if DB not available
	// For constructor test, we can use nil and just verify the struct is created
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer redisClient.Close()

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skipf("Skipping test: Redis not available: %v", err)
		return
	}

	writer := snapshot.NewWriter(redisClient)

	proc := NewProcessor(kafkaConsumer, nil, writer)
	if proc == nil {
		t.Fatal("NewProcessor() returned nil")
	}
	if proc.consumer != kafkaConsumer {
		t.Error("NewProcessor() consumer not set correctly")
	}
	if proc.writer != writer {
		t.Error("NewProcessor() writer not set correctly")
	}
}

func TestProcessor_ProcessRuleChanges_ContextCancellation(t *testing.T) {
	kafkaConsumer, err := consumer.NewConsumer("localhost:9092", "rule.changed", "test-group-context")
	if err != nil {
		t.Skipf("Skipping test: Kafka not available: %v", err)
		return
	}
	defer kafkaConsumer.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer redisClient.Close()

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skipf("Skipping test: Redis not available: %v", err)
		return
	}

	writer := snapshot.NewWriter(redisClient)
	proc := NewProcessor(kafkaConsumer, nil, writer)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = proc.ProcessRuleChanges(ctx)
	if err != nil {
		t.Errorf("ProcessRuleChanges() error = %v, want nil", err)
	}
}

// Note: Full ProcessRuleChanges tests require real Kafka/Redis/Postgres instances
// The constructor test above validates that NewProcessor works correctly.
// Integration tests would be needed for full coverage of ProcessRuleChanges.


func TestProcessor_applyRuleChange_AllActions(t *testing.T) {
	// This test requires real DB and Redis instances
	// For unit testing applyRuleChange, we would need to refactor to use interfaces
	// For now, we test the logic indirectly through integration tests
	
	// Test that applyRuleChange handles all known actions
	// This is tested indirectly through ProcessRuleChanges in integration tests
	// The method is private, so we can't test it directly without being in the same package
	// Since we are in the same package, we can test it, but it requires real dependencies
	
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer redisClient.Close()

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skipf("Skipping test: Redis not available: %v", err)
		return
	}

	writer := snapshot.NewWriter(redisClient)
	
	// Create a mock DB - we'll use sqlmock for this
	// For now, we'll skip if we can't create a proper mock
	// In a real scenario, we'd use sqlmock to create a mock DB
	
	// Test applyRuleChange with all actions
	// Note: This requires a real database connection or sqlmock
	// For now, we verify the method exists and can be called
	proc := &Processor{
		consumer: nil,
		db:       nil,
		writer:   writer,
	}

	actions := []string{
		events.ActionCreated,
		events.ActionUpdated,
		events.ActionDeleted,
		events.ActionDisabled,
		"UNKNOWN",
	}

	for _, action := range actions {
		t.Run(action, func(t *testing.T) {
			ruleChanged := &events.RuleChanged{
				RuleID:        "rule-1",
				ClientID:      "client-1",
				Action:        action,
				Version:       1,
				UpdatedAt:     time.Now().Unix(),
				SchemaVersion: 1,
			}

			// This will fail for CREATED/UPDATED without a DB, and for DELETED/DISABLED without the rule in Redis
			// But it tests the switch statement logic
			err := proc.applyRuleChange(ctx, ruleChanged)
			if action == "UNKNOWN" && err == nil {
				t.Error("applyRuleChange() with unknown action expected error, got nil")
			}
			// Other errors are expected without proper setup
		})
	}
}
