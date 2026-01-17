// Package indexes provides in-memory rule indexes for fast alert matching.
package indexes

import (
	"evaluator/internal/snapshot"
)

// Indexes holds the in-memory rule indexes for fast matching.
// These are built from a snapshot and can be atomically swapped.
type Indexes struct {
	bySeverity map[string][]int // severity -> []ruleInt
	bySource   map[string][]int // source -> []ruleInt
	byName     map[string][]int // name -> []ruleInt
	rules      map[int]snapshot.RuleInfo // ruleInt -> {rule_id, client_id}
}

// NewIndexes creates new indexes from a snapshot.
func NewIndexes(snap *snapshot.Snapshot) *Indexes {
	// Deep copy the maps to ensure we own the data
	bySeverity := make(map[string][]int)
	for k, v := range snap.BySeverity {
		bySeverity[k] = make([]int, len(v))
		copy(bySeverity[k], v)
	}

	bySource := make(map[string][]int)
	for k, v := range snap.BySource {
		bySource[k] = make([]int, len(v))
		copy(bySource[k], v)
	}

	byName := make(map[string][]int)
	for k, v := range snap.ByName {
		byName[k] = make([]int, len(v))
		copy(byName[k], v)
	}

	rules := make(map[int]snapshot.RuleInfo)
	for k, v := range snap.Rules {
		rules[k] = v
	}

	return &Indexes{
		bySeverity: bySeverity,
		bySource:   bySource,
		byName:     byName,
		rules:      rules,
	}
}

// Match finds all rules that match the given alert fields using intersection.
// Supports wildcard "*" values which match any value for that field.
// Returns a map of client_id -> []rule_id for all matching rules.
func (idx *Indexes) Match(severity, source, name string) map[string][]string {
	// Get candidate lists for each field (exact matches)
	severityRules := idx.bySeverity[severity]
	sourceRules := idx.bySource[source]
	nameRules := idx.byName[name]

	// Also get wildcard matches ("*" matches any value)
	wildcardSeverityRules := idx.bySeverity["*"]
	wildcardSourceRules := idx.bySource["*"]
	wildcardNameRules := idx.byName["*"]

	// Combine exact matches with wildcard matches
	allSeverityRules := combineLists(severityRules, wildcardSeverityRules)
	allSourceRules := combineLists(sourceRules, wildcardSourceRules)
	allNameRules := combineLists(nameRules, wildcardNameRules)

	// Find the smallest list to start intersection (minimizes work)
	var candidates []int
	var otherLists [][]int

	if len(allSeverityRules) <= len(allSourceRules) && len(allSeverityRules) <= len(allNameRules) {
		candidates = allSeverityRules
		otherLists = [][]int{allSourceRules, allNameRules}
	} else if len(allSourceRules) <= len(allNameRules) {
		candidates = allSourceRules
		otherLists = [][]int{allSeverityRules, allNameRules}
	} else {
		candidates = allNameRules
		otherLists = [][]int{allSeverityRules, allSourceRules}
	}

	// If any field has no matches, return empty result
	if len(candidates) == 0 {
		return make(map[string][]string)
	}

	// Build sets for the other two lists for fast lookup
	set1 := make(map[int]bool)
	for _, ruleInt := range otherLists[0] {
		set1[ruleInt] = true
	}

	set2 := make(map[int]bool)
	for _, ruleInt := range otherLists[1] {
		set2[ruleInt] = true
	}

	// Intersect: find candidates that exist in both other sets
	matchedRules := make([]int, 0)
	for _, ruleInt := range candidates {
		if set1[ruleInt] && set2[ruleInt] {
			matchedRules = append(matchedRules, ruleInt)
		}
	}

	// Group by client_id
	result := make(map[string][]string)
	for _, ruleInt := range matchedRules {
		ruleInfo, exists := idx.rules[ruleInt]
		if !exists {
			continue // Skip invalid ruleInt
		}
		result[ruleInfo.ClientID] = append(result[ruleInfo.ClientID], ruleInfo.RuleID)
	}

	return result
}

// combineLists combines two lists, removing duplicates.
func combineLists(list1, list2 []int) []int {
	if len(list2) == 0 {
		return list1
	}
	if len(list1) == 0 {
		return list2
	}

	// Use a map to deduplicate
	seen := make(map[int]bool)
	result := make([]int, 0, len(list1)+len(list2))

	for _, v := range list1 {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	for _, v := range list2 {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}

	return result
}

// RuleCount returns the total number of rules in the indexes.
func (idx *Indexes) RuleCount() int {
	return len(idx.rules)
}
