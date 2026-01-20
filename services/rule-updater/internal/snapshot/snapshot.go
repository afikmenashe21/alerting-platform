// Package snapshot handles building and writing rule snapshots to Redis.
package snapshot

import (
	"rule-updater/internal/database"
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

// newEmptySnapshot creates a new empty snapshot with initialized maps.
func newEmptySnapshot() *Snapshot {
	return &Snapshot{
		SchemaVersion: SchemaVersion,
		SeverityDict:  make(map[string]int),
		SourceDict:    make(map[string]int),
		NameDict:      make(map[string]int),
		BySeverity:    make(map[string][]int),
		BySource:      make(map[string][]int),
		ByName:        make(map[string][]int),
		Rules:         make(map[int]RuleInfo),
	}
}

// getMaxDictValue returns the maximum value in a dictionary map.
// Returns 0 if the dictionary is empty.
func getMaxDictValue(dict map[string]int) int {
	max := 0
	for _, v := range dict {
		if v > max {
			max = v
		}
	}
	return max
}

// removeFromIndex removes a ruleInt from an index map and cleans up empty entries.
func removeFromIndex(index map[string][]int, ruleInt int) {
	for key, ruleInts := range index {
		index[key] = removeFromSlice(ruleInts, ruleInt)
		if len(index[key]) == 0 {
			delete(index, key)
		}
	}
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
		snap.SeverityDict[rule.Severity] = getMaxDictValue(snap.SeverityDict) + 1
	}
	if _, exists := snap.SourceDict[rule.Source]; !exists {
		snap.SourceDict[rule.Source] = getMaxDictValue(snap.SourceDict) + 1
	}
	if _, exists := snap.NameDict[rule.Name]; !exists {
		snap.NameDict[rule.Name] = getMaxDictValue(snap.NameDict) + 1
	}

	// Add to inverted indexes
	if snap.BySeverity[rule.Severity] == nil {
		snap.BySeverity[rule.Severity] = make([]int, 0)
	}
	snap.BySeverity[rule.Severity] = append(snap.BySeverity[rule.Severity], ruleInt)

	if snap.BySource[rule.Source] == nil {
		snap.BySource[rule.Source] = make([]int, 0)
	}
	snap.BySource[rule.Source] = append(snap.BySource[rule.Source], ruleInt)

	if snap.ByName[rule.Name] == nil {
		snap.ByName[rule.Name] = make([]int, 0)
	}
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

	// Remove from all indexes using helper function
	removeFromIndex(snap.BySeverity, ruleInt)
	removeFromIndex(snap.BySource, ruleInt)
	removeFromIndex(snap.ByName, ruleInt)

	// Remove from rules map
	delete(snap.Rules, ruleInt)

	// Note: We don't remove from dictionaries even if they're unused
	// This is fine - they're just for compression and unused entries don't hurt

	return nil
}
