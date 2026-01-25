// Package snapshot handles building and writing rule snapshots to Redis.
package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"rule-updater/internal/database"

	"github.com/redis/go-redis/v9"
)

// SnapshotWriter defines the interface for snapshot write operations.
// This interface is implemented by Writer and can be used for testing.
type SnapshotWriter interface {
	WriteSnapshot(ctx context.Context, snapshot *Snapshot) error
	AddRuleDirect(ctx context.Context, rule *database.Rule) error
	RemoveRuleDirect(ctx context.Context, ruleID string) error
	GetVersion(ctx context.Context) (int64, error)
	LoadSnapshot(ctx context.Context) (*Snapshot, error)
}

// Writer handles building and writing snapshots to Redis.
type Writer struct {
	client           *redis.Client
	addRuleScript    *redis.Script
	removeRuleScript *redis.Script
}

// NewWriter creates a new snapshot writer with the given Redis client.
func NewWriter(client *redis.Client) *Writer {
	addScript, removeScript := newLuaScripts()
	return &Writer{
		client:           client,
		addRuleScript:    addScript,
		removeRuleScript: removeScript,
	}
}

// WriteSnapshot writes a snapshot to Redis and increments the version.
// This is an atomic operation: both snapshot and version are updated together.
func (w *Writer) WriteSnapshot(ctx context.Context, snapshot *Snapshot) error {
	// Serialize snapshot to JSON
	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	// Use Redis pipeline to atomically update both snapshot and version
	pipe := w.client.Pipeline()
	pipe.Set(ctx, SnapshotKey, data, 0) // No expiration
	pipe.Incr(ctx, VersionKey)          // Increment version

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to write snapshot to Redis: %w", err)
	}

	// Get the new version for logging
	version, err := w.client.Get(ctx, VersionKey).Int64()
	if err != nil {
		// This shouldn't happen, but log it if it does
		slog.Warn("Failed to get version after write", "error", err)
	} else {
		slog.Info("Snapshot written to Redis",
			"schema_version", snapshot.SchemaVersion,
			"rules_count", len(snapshot.Rules),
			"version", version,
		)
	}

	return nil
}

// GetVersion returns the current rule version from Redis.
// Returns 0 if the version doesn't exist (no rules yet).
func (w *Writer) GetVersion(ctx context.Context) (int64, error) {
	version, err := w.client.Get(ctx, VersionKey).Int64()
	if err == redis.Nil {
		// No version yet, return 0
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get version from Redis: %w", err)
	}
	return version, nil
}

// LoadSnapshot loads the current snapshot from Redis.
// Returns an empty snapshot if no snapshot exists yet.
func (w *Writer) LoadSnapshot(ctx context.Context) (*Snapshot, error) {
	data, err := w.client.Get(ctx, SnapshotKey).Bytes()
	if err == redis.Nil {
		// No snapshot yet, return empty snapshot
		return newEmptySnapshot(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot from Redis: %w", err)
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}

	return &snapshot, nil
}

// AddRuleDirect adds a rule directly to Redis using a Lua script.
// This avoids loading the entire snapshot into Go memory.
func (w *Writer) AddRuleDirect(ctx context.Context, rule *database.Rule) error {
	if !rule.Enabled {
		// Don't add disabled rules
		return nil
	}

	// Execute Lua script to add rule directly in Redis
	// The script handles finding/assigning ruleInt internally
	version, err := w.addRuleScript.Run(ctx, w.client, []string{SnapshotKey, VersionKey},
		rule.RuleID,
		rule.ClientID,
		rule.Severity,
		rule.Source,
		rule.Name,
	).Int64()

	if err != nil {
		return fmt.Errorf("failed to add rule via Lua script: %w", err)
	}

	slog.Info("Rule added directly to Redis",
		"rule_id", rule.RuleID,
		"version", version,
	)

	return nil
}

// RemoveRuleDirect removes a rule directly from Redis using a Lua script.
// This avoids loading the entire snapshot into Go memory.
func (w *Writer) RemoveRuleDirect(ctx context.Context, ruleID string) error {
	// Execute Lua script to remove rule directly in Redis
	version, err := w.removeRuleScript.Run(ctx, w.client, []string{SnapshotKey, VersionKey},
		ruleID,
	).Int64()

	if err != nil {
		return fmt.Errorf("failed to remove rule via Lua script: %w", err)
	}

	if version == 0 {
		// Rule not found, but that's okay
		slog.Info("Rule not found in snapshot (already removed or never existed)",
			"rule_id", ruleID,
		)
		return nil
	}

	slog.Info("Rule removed directly from Redis",
		"rule_id", ruleID,
		"version", version,
	)

	return nil
}
