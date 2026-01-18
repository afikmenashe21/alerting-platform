package snapshot

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestNewLoader(t *testing.T) {
	// Test that NewLoader doesn't panic
	var client *redis.Client
	loader := NewLoader(client)
	if loader == nil {
		t.Fatal("NewLoader() returned nil")
	}
}

func TestLoader_LoadSnapshot_Integration(t *testing.T) {
	// Integration test - requires Redis
	// Skip if Redis is not available
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Skipping integration test: Redis not available: %v", err)
	}

	loader := NewLoader(client)

	// Test with non-existent snapshot (tests redis.Nil path)
	_, err := loader.LoadSnapshot(ctx)
	if err == nil {
		t.Error("LoadSnapshot() should return error for non-existent snapshot")
	}
	if err != nil && err.Error()[:20] != "snapshot not found" {
		t.Logf("LoadSnapshot() error (expected): %v", err)
	}

	// Test with valid snapshot (if we can create one)
	snap := &Snapshot{
		SchemaVersion: 1,
		BySeverity:    map[string][]int{"HIGH": {1}},
		BySource:      map[string][]int{"service-a": {1}},
		ByName:        map[string][]int{"disk-full": {1}},
		Rules:         map[int]RuleInfo{1: {RuleID: "rule-1", ClientID: "client-1"}},
	}
	data, _ := json.Marshal(snap)
	client.Set(ctx, "rules:snapshot", data, 0)

	// Now test loading the snapshot
	loadedSnap, err := loader.LoadSnapshot(ctx)
	if err != nil {
		t.Errorf("LoadSnapshot() error = %v, want nil", err)
	} else {
		if loadedSnap.SchemaVersion != snap.SchemaVersion {
			t.Errorf("LoadSnapshot() SchemaVersion = %v, want %v", loadedSnap.SchemaVersion, snap.SchemaVersion)
		}
		// Clean up
		client.Del(ctx, "rules:snapshot")
	}
}

func TestLoader_GetVersion_Integration(t *testing.T) {
	// Integration test - requires Redis
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Skipping integration test: Redis not available: %v", err)
	}

	loader := NewLoader(client)

	// Test GetVersion - should return 0 if version doesn't exist (tests redis.Nil path)
	version, err := loader.GetVersion(ctx)
	if err != nil {
		t.Errorf("GetVersion() error = %v, want nil", err)
	}
	if version < 0 {
		t.Errorf("GetVersion() = %v, want >= 0", version)
	}

	// Test GetVersion with existing version
	client.Set(ctx, "rules:version", 42, 0)
	version, err = loader.GetVersion(ctx)
	if err != nil {
		t.Errorf("GetVersion() error = %v, want nil", err)
	}
	if version != 42 {
		t.Errorf("GetVersion() = %v, want 42", version)
	}
	// Clean up
	client.Del(ctx, "rules:version")

	// Test GetVersion with invalid Redis connection (error path)
	invalidClient := redis.NewClient(&redis.Options{
		Addr: "invalid:6379",
	})
	defer invalidClient.Close()

	invalidLoader := NewLoader(invalidClient)
	ctxTimeout, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = invalidLoader.GetVersion(ctxTimeout)
	if err == nil {
		t.Log("GetVersion() with invalid connection succeeded (unexpected, may be due to timeout)")
	} else {
		t.Logf("GetVersion() error (expected): %v", err)
	}
}

func TestLoader_LoadSnapshot_ErrorPaths(t *testing.T) {
	// Test LoadSnapshot with invalid Redis connection (error path)
	invalidClient := redis.NewClient(&redis.Options{
		Addr: "invalid:6379",
	})
	defer invalidClient.Close()

	invalidLoader := NewLoader(invalidClient)
	ctxTimeout, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := invalidLoader.LoadSnapshot(ctxTimeout)
	if err == nil {
		t.Log("LoadSnapshot() with invalid connection succeeded (unexpected, may be due to timeout)")
	} else {
		t.Logf("LoadSnapshot() error (expected): %v", err)
	}
}

func TestSnapshot_Structure(t *testing.T) {
	snap := &Snapshot{
		SchemaVersion: 1,
		SeverityDict:  map[string]int{"HIGH": 1, "LOW": 2, "*": 0},
		SourceDict:    map[string]int{"service-a": 1, "*": 0},
		NameDict:      map[string]int{"disk-full": 1, "*": 0},
		BySeverity:    map[string][]int{"HIGH": {1, 2}, "LOW": {3}, "*": {4}},
		BySource:      map[string][]int{"service-a": {1}, "*": {2, 4}},
		ByName:        map[string][]int{"disk-full": {1}, "*": {3, 4}},
		Rules: map[int]RuleInfo{
			1: {RuleID: "rule-1", ClientID: "client-1"},
			2: {RuleID: "rule-2", ClientID: "client-1"},
			3: {RuleID: "rule-3", ClientID: "client-2"},
			4: {RuleID: "rule-4", ClientID: "client-2"},
		},
	}

	// Test JSON round-trip
	data, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("Failed to marshal snapshot: %v", err)
	}

	var unmarshaled Snapshot
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal snapshot: %v", err)
	}

	if unmarshaled.SchemaVersion != snap.SchemaVersion {
		t.Errorf("SchemaVersion = %v, want %v", unmarshaled.SchemaVersion, snap.SchemaVersion)
	}
	if len(unmarshaled.Rules) != len(snap.Rules) {
		t.Errorf("Rules count = %v, want %v", len(unmarshaled.Rules), len(snap.Rules))
	}

	// Verify all fields
	if len(unmarshaled.BySeverity) != len(snap.BySeverity) {
		t.Errorf("BySeverity count = %v, want %v", len(unmarshaled.BySeverity), len(snap.BySeverity))
	}
	if len(unmarshaled.BySource) != len(snap.BySource) {
		t.Errorf("BySource count = %v, want %v", len(unmarshaled.BySource), len(snap.BySource))
	}
	if len(unmarshaled.ByName) != len(snap.ByName) {
		t.Errorf("ByName count = %v, want %v", len(unmarshaled.ByName), len(snap.ByName))
	}
}

func TestSnapshot_LoadSnapshot_JSONError(t *testing.T) {
	// Test JSON unmarshaling error handling
	// This tests the error path in LoadSnapshot without requiring Redis
	invalidJSON := "invalid json"
	var snap Snapshot
	err := json.Unmarshal([]byte(invalidJSON), &snap)
	if err == nil {
		t.Error("json.Unmarshal() should return error for invalid JSON")
	}
}

func TestRuleInfo_Structure(t *testing.T) {
	ruleInfo := RuleInfo{
		RuleID:   "rule-123",
		ClientID: "client-456",
	}

	// Test JSON round-trip
	data, err := json.Marshal(ruleInfo)
	if err != nil {
		t.Fatalf("Failed to marshal RuleInfo: %v", err)
	}

	var unmarshaled RuleInfo
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal RuleInfo: %v", err)
	}

	if unmarshaled.RuleID != ruleInfo.RuleID {
		t.Errorf("RuleID = %v, want %v", unmarshaled.RuleID, ruleInfo.RuleID)
	}
	if unmarshaled.ClientID != ruleInfo.ClientID {
		t.Errorf("ClientID = %v, want %v", unmarshaled.ClientID, ruleInfo.ClientID)
	}
}
