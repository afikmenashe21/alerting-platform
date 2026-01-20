// Package snapshot handles building and writing rule snapshots to Redis.
package snapshot

import (
	"rule-updater/internal/database"
)

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

// addToIndex adds a ruleInt to an index map, initializing the slice if needed.
func addToIndex(index map[string][]int, key string, ruleInt int) {
	if index[key] == nil {
		index[key] = make([]int, 0)
	}
	index[key] = append(index[key], ruleInt)
}

// addToDictionaries adds rule values to dictionaries if they don't exist.
func (snap *Snapshot) addToDictionaries(severity, source, name string) {
	if _, exists := snap.SeverityDict[severity]; !exists {
		snap.SeverityDict[severity] = getMaxDictValue(snap.SeverityDict) + 1
	}
	if _, exists := snap.SourceDict[source]; !exists {
		snap.SourceDict[source] = getMaxDictValue(snap.SourceDict) + 1
	}
	if _, exists := snap.NameDict[name]; !exists {
		snap.NameDict[name] = getMaxDictValue(snap.NameDict) + 1
	}
}

// addToIndexes adds a ruleInt to all inverted indexes.
func (snap *Snapshot) addToIndexes(severity, source, name string, ruleInt int) {
	addToIndex(snap.BySeverity, severity, ruleInt)
	addToIndex(snap.BySource, source, ruleInt)
	addToIndex(snap.ByName, name, ruleInt)
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
	snap.addToDictionaries(rule.Severity, rule.Source, rule.Name)

	// Add to inverted indexes
	snap.addToIndexes(rule.Severity, rule.Source, rule.Name, ruleInt)

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
