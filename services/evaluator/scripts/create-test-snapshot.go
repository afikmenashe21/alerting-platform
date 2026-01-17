// Package main creates a test rule snapshot in Redis for testing the evaluator.
// This is a temporary utility until rule-updater is fully implemented.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

const (
	SnapshotKey = "rules:snapshot"
	VersionKey = "rules:version"
)

// Snapshot represents the serialized rule indexes.
type Snapshot struct {
	SchemaVersion int                    `json:"schema_version"`
	SeverityDict  map[string]int         `json:"severity_dict"`
	SourceDict    map[string]int          `json:"source_dict"`
	NameDict      map[string]int          `json:"name_dict"`
	BySeverity    map[string][]int        `json:"by_severity"`
	BySource      map[string][]int        `json:"by_source"`
	ByName        map[string][]int        `json:"by_name"`
	Rules         map[int]RuleInfo         `json:"rules"`
}

// RuleInfo contains the rule ID and client ID for a given ruleInt.
type RuleInfo struct {
	RuleID   string `json:"rule_id"`
	ClientID string `json:"client_id"`
}

func main() {
	redisAddr := "localhost:6379"
	if len(os.Args) > 1 {
		redisAddr = os.Args[1]
	}

	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	defer client.Close()

	ctx := context.Background()

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Create test rules
	// Rule 1: HIGH severity, api source, timeout name -> client-1
	// Rule 2: MEDIUM severity, db source, error name -> client-1
	// Rule 3: HIGH severity, api source, timeout name -> client-2 (same as rule 1 but different client)

	// Build dictionaries (string -> int mapping for compression)
	severityDict := map[string]int{
		"HIGH":   1,
		"MEDIUM": 2,
		"LOW":    3,
	}
	sourceDict := map[string]int{
		"api": 1,
		"db":  2,
	}
	nameDict := map[string]int{
		"timeout": 1,
		"error":   2,
	}

	// Build inverted indexes
	// ruleInt 1: HIGH + api + timeout -> client-1
	// ruleInt 2: MEDIUM + db + error -> client-1
	// ruleInt 3: HIGH + api + timeout -> client-2

	bySeverity := map[string][]int{
		"HIGH":    {1, 3}, // ruleInts 1 and 3 match HIGH
		"MEDIUM":  {2},    // ruleInt 2 matches MEDIUM
		"LOW":     {},     // no rules for LOW
	}

	bySource := map[string][]int{
		"api": {1, 3}, // ruleInts 1 and 3 match api
		"db":  {2},    // ruleInt 2 matches db
	}

	byName := map[string][]int{
		"timeout": {1, 3}, // ruleInts 1 and 3 match timeout
		"error":   {2},    // ruleInt 2 matches error
	}

	// Build rule mapping
	rules := map[int]RuleInfo{
		1: {RuleID: "rule-001", ClientID: "client-1"},
		2: {RuleID: "rule-002", ClientID: "client-1"},
		3: {RuleID: "rule-003", ClientID: "client-2"},
	}

	snapshot := Snapshot{
		SchemaVersion: 1,
		SeverityDict:  severityDict,
		SourceDict:    sourceDict,
		NameDict:      nameDict,
		BySeverity:    bySeverity,
		BySource:      bySource,
		ByName:        byName,
		Rules:         rules,
	}

	// Serialize to JSON
	data, err := json.Marshal(snapshot)
	if err != nil {
		log.Fatalf("Failed to marshal snapshot: %v", err)
	}

	// Write to Redis
	if err := client.Set(ctx, SnapshotKey, data, 0).Err(); err != nil {
		log.Fatalf("Failed to write snapshot to Redis: %v", err)
	}

	// Set version
	if err := client.Set(ctx, VersionKey, 1, 0).Err(); err != nil {
		log.Fatalf("Failed to write version to Redis: %v", err)
	}

	fmt.Printf("✅ Created test rule snapshot in Redis\n")
	fmt.Printf("   Snapshot key: %s\n", SnapshotKey)
	fmt.Printf("   Version key: %s\n", VersionKey)
	fmt.Printf("   Version: 1\n")
	fmt.Printf("   Rules: 3\n")
	fmt.Printf("\nTest rules:\n")
	fmt.Printf("  - Rule 1: HIGH + api + timeout -> client-1\n")
	fmt.Printf("  - Rule 2: MEDIUM + db + error -> client-1\n")
	fmt.Printf("  - Rule 3: HIGH + api + timeout -> client-2\n")
	fmt.Printf("\nYou can now run the evaluator!\n")
}
