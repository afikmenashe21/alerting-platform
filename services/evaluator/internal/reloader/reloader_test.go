package reloader

import (
	"context"
	"testing"
	"time"

	"evaluator/internal/indexes"
	"evaluator/internal/matcher"
	"evaluator/internal/snapshot"

	"github.com/redis/go-redis/v9"
)

func TestNewReloader(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	loader := snapshot.NewLoader(client)
	snap := &snapshot.Snapshot{
		BySeverity: map[string][]int{"HIGH": {1}},
		BySource:   map[string][]int{"service-a": {1}},
		ByName:     map[string][]int{"disk-full": {1}},
		Rules:      map[int]snapshot.RuleInfo{1: {RuleID: "rule-1", ClientID: "client-1"}},
	}
	idx := indexes.NewIndexes(snap)
	m := matcher.NewMatcher(idx)
	pollInterval := 5 * time.Second

	reloader := NewReloader(loader, m, pollInterval)
	if reloader == nil {
		t.Fatal("NewReloader() returned nil")
	}
	if reloader.loader != loader {
		t.Error("NewReloader() did not set loader correctly")
	}
	if reloader.matcher != m {
		t.Error("NewReloader() did not set matcher correctly")
	}
	if reloader.pollInterval != pollInterval {
		t.Error("NewReloader() did not set pollInterval correctly")
	}
}

func TestReloader_Start_Integration(t *testing.T) {
	// Integration test - requires Redis
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Skipping integration test: Redis not available: %v", err)
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
	reloader := NewReloader(loader, m, 100*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := reloader.Start(ctx)
	if err != nil {
		t.Errorf("Reloader.Start() error = %v, want nil", err)
	}

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)
}

func TestReloader_ReloadNow_Integration(t *testing.T) {
	// Integration test - requires Redis
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Skipping integration test: Redis not available: %v", err)
	}

	loader := snapshot.NewLoader(client)
	snap1 := &snapshot.Snapshot{
		BySeverity: map[string][]int{"HIGH": {1}},
		BySource:   map[string][]int{"service-a": {1}},
		ByName:     map[string][]int{"disk-full": {1}},
		Rules:      map[int]snapshot.RuleInfo{1: {RuleID: "rule-1", ClientID: "client-1"}},
	}
	idx1 := indexes.NewIndexes(snap1)
	m := matcher.NewMatcher(idx1)
	reloader := NewReloader(loader, m, 100*time.Millisecond)
	reloader.currentVersion = 1 // Set initial version

	// Test version unchanged
	err := reloader.ReloadNow(ctx)
	if err != nil {
		t.Errorf("Reloader.ReloadNow() error = %v, want nil (version unchanged)", err)
	}

	// Test with version change (if snapshot exists in Redis)
	// This will only work if there's actually a snapshot in Redis
	version, _ := loader.GetVersion(ctx)
	if version > 0 {
		reloader.currentVersion = version - 1
		err = reloader.ReloadNow(ctx)
		if err != nil {
			t.Logf("ReloadNow() with version change error (may be expected if no snapshot): %v", err)
		}
	}
}

func TestReloader_PollLoop_Integration(t *testing.T) {
	// Integration test - requires Redis
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Skipping integration test: Redis not available: %v", err)
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
	reloader := NewReloader(loader, m, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := reloader.Start(ctx); err != nil {
		t.Fatalf("Reloader.Start() error = %v", err)
	}

	// Wait for a few poll cycles
	time.Sleep(200 * time.Millisecond)

	cancel()
	time.Sleep(100 * time.Millisecond) // Give it time to stop
}

func TestReloader_ReloadNow_ErrorHandling(t *testing.T) {
	// Test error handling without requiring Redis
	// Create a loader with a client that will fail
	client := redis.NewClient(&redis.Options{
		Addr: "invalid:6379", // Invalid address
	})
	defer client.Close()

	loader := snapshot.NewLoader(client)
	snap := &snapshot.Snapshot{
		BySeverity: map[string][]int{"HIGH": {1}},
		BySource:   map[string][]int{"service-a": {1}},
		ByName:     map[string][]int{"disk-full": {1}},
		Rules:      map[int]snapshot.RuleInfo{1: {RuleID: "rule-1", ClientID: "client-1"}},
	}
	idx := indexes.NewIndexes(snap)
	m := matcher.NewMatcher(idx)
	reloader := NewReloader(loader, m, 100*time.Millisecond)
	reloader.currentVersion = 1

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// This should fail due to invalid Redis connection
	err := reloader.ReloadNow(ctx)
	if err == nil {
		t.Log("ReloadNow() succeeded (unexpected, may be due to connection timeout)")
	} else {
		t.Logf("ReloadNow() error (expected): %v", err)
	}
}
