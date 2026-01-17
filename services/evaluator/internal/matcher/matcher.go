// Package matcher provides alert matching functionality using rule indexes.
package matcher

import (
	"evaluator/internal/indexes"
	"sync"
)

// Matcher provides thread-safe access to rule indexes for matching alerts.
// It supports atomic swapping of indexes when rules are updated.
type Matcher struct {
	mu      sync.RWMutex
	indexes *indexes.Indexes
}

// NewMatcher creates a new matcher with the given initial indexes.
func NewMatcher(idx *indexes.Indexes) *Matcher {
	return &Matcher{
		indexes: idx,
	}
}

// Match finds all rules that match the given alert fields.
// Returns a map of client_id -> []rule_id for all matching rules.
// Thread-safe: uses read lock for concurrent access.
func (m *Matcher) Match(severity, source, name string) map[string][]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.indexes.Match(severity, source, name)
}

// UpdateIndexes atomically swaps the indexes with new ones.
// Thread-safe: uses write lock to ensure atomic update.
func (m *Matcher) UpdateIndexes(idx *indexes.Indexes) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.indexes = idx
}

// RuleCount returns the current number of rules in the indexes.
func (m *Matcher) RuleCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.indexes.RuleCount()
}
