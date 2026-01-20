// Package snapshot handles building and writing rule snapshots to Redis.
package snapshot

import (
	"rule-updater/internal/database"
)

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
