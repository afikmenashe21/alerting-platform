// Package snapshot handles loading and deserializing rule snapshots from Redis.
package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

const (
	// SnapshotKey is the Redis key where the rule snapshot is stored.
	SnapshotKey = "rules:snapshot"
	// VersionKey is the Redis key where the rule version is stored.
	VersionKey = "rules:version"
)

// Snapshot represents the serialized rule indexes loaded from Redis.
type Snapshot struct {
	SchemaVersion int                    `json:"schema_version"`
	SeverityDict  map[string]int         `json:"severity_dict"`
	SourceDict    map[string]int          `json:"source_dict"`
	NameDict      map[string]int          `json:"name_dict"`
	BySeverity    map[string][]int        `json:"by_severity"` // severity -> []ruleInt
	BySource      map[string][]int        `json:"by_source"`   // source -> []ruleInt
	ByName        map[string][]int        `json:"by_name"`     // name -> []ruleInt
	Rules         map[int]RuleInfo         `json:"rules"`       // ruleInt -> {rule_id, client_id}
}

// RuleInfo contains the rule ID and client ID for a given ruleInt.
type RuleInfo struct {
	RuleID   string `json:"rule_id"`
	ClientID string `json:"client_id"`
}

// Loader handles loading snapshots from Redis.
type Loader struct {
	client *redis.Client
}

// NewLoader creates a new snapshot loader with the given Redis client.
func NewLoader(client *redis.Client) *Loader {
	return &Loader{
		client: client,
	}
}

// LoadSnapshot loads the rule snapshot from Redis and deserializes it.
// Returns an error if the snapshot doesn't exist or deserialization fails.
func (l *Loader) LoadSnapshot(ctx context.Context) (*Snapshot, error) {
	data, err := l.client.Get(ctx, SnapshotKey).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("snapshot not found in Redis (key: %s)", SnapshotKey)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot from Redis: %w", err)
	}

	var snapshot Snapshot
	if err := json.Unmarshal([]byte(data), &snapshot); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}

	slog.Info("Loaded rule snapshot from Redis",
		"schema_version", snapshot.SchemaVersion,
		"rules_count", len(snapshot.Rules),
	)

	return &snapshot, nil
}

// GetVersion returns the current rule version from Redis.
// Returns 0 if the version doesn't exist (no rules yet).
func (l *Loader) GetVersion(ctx context.Context) (int64, error) {
	version, err := l.client.Get(ctx, VersionKey).Int64()
	if err == redis.Nil {
		// No version yet, return 0
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get version from Redis: %w", err)
	}
	return version, nil
}
