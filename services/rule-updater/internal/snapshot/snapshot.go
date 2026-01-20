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

