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

const (
	// SnapshotKey is the Redis key where the rule snapshot is stored.
	SnapshotKey = "rules:snapshot"
	// VersionKey is the Redis key where the rule version is stored.
	VersionKey = "rules:version"
	// SchemaVersion is the current schema version for the snapshot format.
	SchemaVersion = 1
)

// Snapshot represents the serialized rule indexes written to Redis.
// This matches the structure expected by the evaluator.
type Snapshot struct {
	SchemaVersion int                    `json:"schema_version"`
	SeverityDict  map[string]int         `json:"severity_dict"`
	SourceDict    map[string]int         `json:"source_dict"`
	NameDict      map[string]int         `json:"name_dict"`
	BySeverity    map[string][]int        `json:"by_severity"` // severity -> []ruleInt
	BySource      map[string][]int        `json:"by_source"`   // source -> []ruleInt
	ByName        map[string][]int        `json:"by_name"`     // name -> []ruleInt
	Rules         map[int]RuleInfo        `json:"rules"`       // ruleInt -> {rule_id, client_id}
}

// RuleInfo contains the rule ID and client ID for a given ruleInt.
type RuleInfo struct {
	RuleID   string `json:"rule_id"`
	ClientID string `json:"client_id"`
}

// Writer handles building and writing snapshots to Redis.
type Writer struct {
	client *redis.Client
	// Lua scripts for direct Redis updates
	addRuleScript    *redis.Script
	removeRuleScript *redis.Script
}

// Lua scripts for direct Redis updates
const (
	// addRuleScript adds or updates a rule in the snapshot JSON directly in Redis
	addRuleScript = `
		local snapshot_key = KEYS[1]
		local version_key = KEYS[2]
		local rule_id = ARGV[1]
		local client_id = ARGV[2]
		local severity = ARGV[3]
		local source = ARGV[4]
		local name = ARGV[5]
		
		-- Load current snapshot
		local snapshot_json = redis.call('GET', snapshot_key)
		if not snapshot_json then
			-- Create empty snapshot
			snapshot_json = '{"schema_version":1,"severity_dict":{},"source_dict":{},"name_dict":{},"by_severity":{},"by_source":{},"by_name":{},"rules":{}}'
		end
		
		local snapshot = cjson.decode(snapshot_json)
		
		-- Find existing ruleInt for this rule_id
		local existing_rule_int = nil
		for rule_int_key, rule_info in pairs(snapshot.rules) do
			if rule_info.rule_id == rule_id then
				existing_rule_int = tonumber(rule_int_key)
				break
			end
		end
		
		-- Determine rule_int to use
		local rule_int
		if existing_rule_int then
			rule_int = existing_rule_int
			-- Remove from indexes first
			for sev, rule_ints in pairs(snapshot.by_severity) do
				for i = #rule_ints, 1, -1 do
					if rule_ints[i] == existing_rule_int then
						table.remove(rule_ints, i)
					end
				end
				if #rule_ints == 0 then
					snapshot.by_severity[sev] = nil
				end
			end
			for src, rule_ints in pairs(snapshot.by_source) do
				for i = #rule_ints, 1, -1 do
					if rule_ints[i] == existing_rule_int then
						table.remove(rule_ints, i)
					end
				end
				if #rule_ints == 0 then
					snapshot.by_source[src] = nil
				end
			end
			for nm, rule_ints in pairs(snapshot.by_name) do
				for i = #rule_ints, 1, -1 do
					if rule_ints[i] == existing_rule_int then
						table.remove(rule_ints, i)
					end
				end
				if #rule_ints == 0 then
					snapshot.by_name[nm] = nil
				end
			end
		else
			-- Find next available rule_int
			local max_rule_int = 0
			for rule_int_key, _ in pairs(snapshot.rules) do
				local rint = tonumber(rule_int_key)
				if rint and rint > max_rule_int then
					max_rule_int = rint
				end
			end
			rule_int = max_rule_int + 1
		end
		
		-- Add to dictionaries if needed
		if not snapshot.severity_dict[severity] then
			local max_sev = 0
			for _, v in pairs(snapshot.severity_dict) do
				if v > max_sev then max_sev = v end
			end
			snapshot.severity_dict[severity] = max_sev + 1
		end
		if not snapshot.source_dict[source] then
			local max_src = 0
			for _, v in pairs(snapshot.source_dict) do
				if v > max_src then max_src = v end
			end
			snapshot.source_dict[source] = max_src + 1
		end
		if not snapshot.name_dict[name] then
			local max_name = 0
			for _, v in pairs(snapshot.name_dict) do
				if v > max_name then max_name = v end
			end
			snapshot.name_dict[name] = max_name + 1
		end
		
		-- Add to indexes
		if not snapshot.by_severity[severity] then
			snapshot.by_severity[severity] = {}
		end
		table.insert(snapshot.by_severity[severity], rule_int)
		
		if not snapshot.by_source[source] then
			snapshot.by_source[source] = {}
		end
		table.insert(snapshot.by_source[source], rule_int)
		
		if not snapshot.by_name[name] then
			snapshot.by_name[name] = {}
		end
		table.insert(snapshot.by_name[name], rule_int)
		
		-- Add to rules map
		snapshot.rules[tostring(rule_int)] = {
			rule_id = rule_id,
			client_id = client_id
		}
		
		-- Write back and increment version
		local updated_json = cjson.encode(snapshot)
		redis.call('SET', snapshot_key, updated_json)
		return redis.call('INCR', version_key)
	`

	// removeRuleScript removes a rule from the snapshot JSON directly in Redis
	removeRuleScript = `
		local snapshot_key = KEYS[1]
		local version_key = KEYS[2]
		local rule_id = ARGV[1]
		
		-- Load current snapshot
		local snapshot_json = redis.call('GET', snapshot_key)
		if not snapshot_json then
			return 0
		end
		
		local snapshot = cjson.decode(snapshot_json)
		
		-- Find ruleInt for this rule_id
		local rule_int = nil
		for rule_int_key, rule_info in pairs(snapshot.rules) do
			if rule_info.rule_id == rule_id then
				rule_int = tonumber(rule_int_key)
				break
			end
		end
		
		if not rule_int then
			return 0
		end
		
		-- Remove from indexes
		for sev, rule_ints in pairs(snapshot.by_severity) do
			for i = #rule_ints, 1, -1 do
				if rule_ints[i] == rule_int then
					table.remove(rule_ints, i)
				end
			end
			if #rule_ints == 0 then
				snapshot.by_severity[sev] = nil
			end
		end
		for src, rule_ints in pairs(snapshot.by_source) do
			for i = #rule_ints, 1, -1 do
				if rule_ints[i] == rule_int then
					table.remove(rule_ints, i)
				end
			end
			if #rule_ints == 0 then
				snapshot.by_source[src] = nil
			end
		end
		for nm, rule_ints in pairs(snapshot.by_name) do
			for i = #rule_ints, 1, -1 do
				if rule_ints[i] == rule_int then
					table.remove(rule_ints, i)
				end
			end
			if #rule_ints == 0 then
				snapshot.by_name[nm] = nil
			end
		end
		
		-- Remove from rules map
		snapshot.rules[tostring(rule_int)] = nil
		
		-- Write back and increment version
		local updated_json = cjson.encode(snapshot)
		redis.call('SET', snapshot_key, updated_json)
		return redis.call('INCR', version_key)
	`
)

// NewWriter creates a new snapshot writer with the given Redis client.
func NewWriter(client *redis.Client) *Writer {
	return &Writer{
		client:           client,
		addRuleScript:    redis.NewScript(addRuleScript),
		removeRuleScript: redis.NewScript(removeRuleScript),
	}
}

// BuildSnapshot builds a snapshot from a list of enabled rules.
// It creates dictionaries for compression and inverted indexes for fast matching.
func BuildSnapshot(rules []*database.Rule) *Snapshot {
	// Build dictionaries (string -> int mapping for compression)
	severityDict := make(map[string]int)
	sourceDict := make(map[string]int)
	nameDict := make(map[string]int)

	// Track unique values and assign integers
	severityInt := 1
	sourceInt := 1
	nameInt := 1

	// First pass: build dictionaries
	for _, rule := range rules {
		if _, exists := severityDict[rule.Severity]; !exists {
			severityDict[rule.Severity] = severityInt
			severityInt++
		}
		if _, exists := sourceDict[rule.Source]; !exists {
			sourceDict[rule.Source] = sourceInt
			sourceInt++
		}
		if _, exists := nameDict[rule.Name]; !exists {
			nameDict[rule.Name] = nameInt
			nameInt++
		}
	}

	// Build inverted indexes and rule mapping
	// ruleInt is a unique integer assigned to each rule (1, 2, 3, ...)
	bySeverity := make(map[string][]int)
	bySource := make(map[string][]int)
	byName := make(map[string][]int)
	rulesMap := make(map[int]RuleInfo)

	ruleInt := 1
	for _, rule := range rules {
		// Add to inverted indexes
		bySeverity[rule.Severity] = append(bySeverity[rule.Severity], ruleInt)
		bySource[rule.Source] = append(bySource[rule.Source], ruleInt)
		byName[rule.Name] = append(byName[rule.Name], ruleInt)

		// Store rule info
		rulesMap[ruleInt] = RuleInfo{
			RuleID:   rule.RuleID,
			ClientID: rule.ClientID,
		}

		ruleInt++
	}

	return &Snapshot{
		SchemaVersion: SchemaVersion,
		SeverityDict:  severityDict,
		SourceDict:    sourceDict,
		NameDict:      nameDict,
		BySeverity:    bySeverity,
		BySource:      bySource,
		ByName:        byName,
		Rules:         rulesMap,
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
// Returns nil if no snapshot exists yet.
func (w *Writer) LoadSnapshot(ctx context.Context) (*Snapshot, error) {
	data, err := w.client.Get(ctx, SnapshotKey).Bytes()
	if err == redis.Nil {
		// No snapshot yet, return empty snapshot
		return &Snapshot{
			SchemaVersion: SchemaVersion,
			SeverityDict:  make(map[string]int),
			SourceDict:    make(map[string]int),
			NameDict:      make(map[string]int),
			BySeverity:    make(map[string][]int),
			BySource:      make(map[string][]int),
			ByName:        make(map[string][]int),
			Rules:         make(map[int]RuleInfo),
		}, nil
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

// findRuleInt finds the ruleInt for a given rule_id in the snapshot.
// Returns 0 if not found.
func (snap *Snapshot) findRuleInt(ruleID string) int {
	for ruleInt, ruleInfo := range snap.Rules {
		if ruleInfo.RuleID == ruleID {
			return ruleInt
		}
	}
	return 0
}

// getNextRuleInt returns the next available ruleInt.
func (snap *Snapshot) getNextRuleInt() int {
	maxRuleInt := 0
	for ruleInt := range snap.Rules {
		if ruleInt > maxRuleInt {
			maxRuleInt = ruleInt
		}
	}
	return maxRuleInt + 1
}

// removeFromSlice removes a value from a slice of integers.
func removeFromSlice(slice []int, value int) []int {
	result := make([]int, 0, len(slice))
	for _, v := range slice {
		if v != value {
			result = append(result, v)
		}
	}
	return result
}

// AddRule adds a new rule to the snapshot.
// If the rule already exists (by rule_id), it updates it instead.
func (snap *Snapshot) AddRule(rule *database.Rule) error {
	// Check if rule already exists
	existingRuleInt := snap.findRuleInt(rule.RuleID)
	if existingRuleInt > 0 {
		// Rule exists, update it instead
		return snap.UpdateRule(rule)
	}

	// Ensure rule is enabled
	if !rule.Enabled {
		// Don't add disabled rules
		return nil
	}

	// Get next ruleInt
	ruleInt := snap.getNextRuleInt()

	// Add to dictionaries if needed
	if _, exists := snap.SeverityDict[rule.Severity]; !exists {
		// Find next severity int
		maxSeverityInt := 0
		for _, v := range snap.SeverityDict {
			if v > maxSeverityInt {
				maxSeverityInt = v
			}
		}
		snap.SeverityDict[rule.Severity] = maxSeverityInt + 1
	}
	if _, exists := snap.SourceDict[rule.Source]; !exists {
		// Find next source int
		maxSourceInt := 0
		for _, v := range snap.SourceDict {
			if v > maxSourceInt {
				maxSourceInt = v
			}
		}
		snap.SourceDict[rule.Source] = maxSourceInt + 1
	}
	if _, exists := snap.NameDict[rule.Name]; !exists {
		// Find next name int
		maxNameInt := 0
		for _, v := range snap.NameDict {
			if v > maxNameInt {
				maxNameInt = v
			}
		}
		snap.NameDict[rule.Name] = maxNameInt + 1
	}

	// Add to inverted indexes
	snap.BySeverity[rule.Severity] = append(snap.BySeverity[rule.Severity], ruleInt)
	snap.BySource[rule.Source] = append(snap.BySource[rule.Source], ruleInt)
	snap.ByName[rule.Name] = append(snap.ByName[rule.Name], ruleInt)

	// Store rule info
	snap.Rules[ruleInt] = RuleInfo{
		RuleID:   rule.RuleID,
		ClientID: rule.ClientID,
	}

	return nil
}

// UpdateRule updates an existing rule in the snapshot.
// If the rule doesn't exist, it adds it instead.
func (snap *Snapshot) UpdateRule(rule *database.Rule) error {
	// Find existing ruleInt
	ruleInt := snap.findRuleInt(rule.RuleID)
	if ruleInt == 0 {
		// Rule doesn't exist, add it instead
		return snap.AddRule(rule)
	}

	// If rule is now disabled, remove it
	if !rule.Enabled {
		return snap.RemoveRule(rule.RuleID)
	}

	// Remove the old rule completely and re-add it with new values.
	// This is safe because we know the rule_id, and RemoveRule will clean up
	// all index entries (it searches through all indexes to find and remove the ruleInt).
	snap.RemoveRule(rule.RuleID)
	return snap.AddRule(rule)
}

// RemoveRule removes a rule from the snapshot by rule_id.
func (snap *Snapshot) RemoveRule(ruleID string) error {
	// Find ruleInt
	ruleInt := snap.findRuleInt(ruleID)
	if ruleInt == 0 {
		// Rule not found, nothing to remove
		return nil
	}

	// Remove from severity index by searching through all severity entries
	// The ruleInt will only be in the list matching the rule's severity value
	for severity, ruleInts := range snap.BySeverity {
		snap.BySeverity[severity] = removeFromSlice(ruleInts, ruleInt)
		if len(snap.BySeverity[severity]) == 0 {
			delete(snap.BySeverity, severity)
		}
	}

	// Remove from source index
	for source, ruleInts := range snap.BySource {
		snap.BySource[source] = removeFromSlice(ruleInts, ruleInt)
		if len(snap.BySource[source]) == 0 {
			delete(snap.BySource, source)
		}
	}

	// Remove from name index
	for name, ruleInts := range snap.ByName {
		snap.ByName[name] = removeFromSlice(ruleInts, ruleInt)
		if len(snap.ByName[name]) == 0 {
			delete(snap.ByName, name)
		}
	}

	// Remove from rules map
	delete(snap.Rules, ruleInt)

	// Note: We don't remove from dictionaries even if they're unused
	// This is fine - they're just for compression and unused entries don't hurt

	return nil
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
