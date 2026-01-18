package matcher

import (
	"evaluator/internal/indexes"
	"evaluator/internal/snapshot"
	"sync"
	"testing"
)

func TestNewMatcher(t *testing.T) {
	snap := &snapshot.Snapshot{
		BySeverity: map[string][]int{"HIGH": {1}},
		BySource:   map[string][]int{"service-a": {1}},
		ByName:     map[string][]int{"disk-full": {1}},
		Rules:      map[int]snapshot.RuleInfo{1: {RuleID: "rule-1", ClientID: "client-1"}},
	}
	idx := indexes.NewIndexes(snap)

	matcher := NewMatcher(idx)
	if matcher == nil {
		t.Fatal("NewMatcher() returned nil")
	}

	// Verify initial state
	if matcher.RuleCount() != 1 {
		t.Errorf("NewMatcher() RuleCount() = %v, want 1", matcher.RuleCount())
	}
}

func TestMatcher_Match(t *testing.T) {
	snap := &snapshot.Snapshot{
		SchemaVersion: 1,
		BySeverity:    map[string][]int{"HIGH": {1}, "LOW": {2}},
		BySource:      map[string][]int{"service-a": {1}, "service-b": {2}},
		ByName:        map[string][]int{"disk-full": {1}, "cpu-high": {2}},
		Rules: map[int]snapshot.RuleInfo{
			1: {RuleID: "rule-1", ClientID: "client-1"},
			2: {RuleID: "rule-2", ClientID: "client-2"},
		},
	}
	idx := indexes.NewIndexes(snap)
	matcher := NewMatcher(idx)

	tests := []struct {
		name          string
		severity      string
		source        string
		nameField     string
		wantClientIDs []string
	}{
		{
			name:          "match rule-1",
			severity:      "HIGH",
			source:        "service-a",
			nameField:     "disk-full",
			wantClientIDs: []string{"client-1"},
		},
		{
			name:          "match rule-2",
			severity:      "LOW",
			source:        "service-b",
			nameField:     "cpu-high",
			wantClientIDs: []string{"client-2"},
		},
		{
			name:          "no match",
			severity:      "MEDIUM",
			source:        "service-c",
			nameField:     "unknown",
			wantClientIDs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.Match(tt.severity, tt.source, tt.nameField)

			if len(result) != len(tt.wantClientIDs) {
				t.Errorf("Match() returned %d clients, want %d", len(result), len(tt.wantClientIDs))
			}

			for _, wantID := range tt.wantClientIDs {
				if _, exists := result[wantID]; !exists {
					t.Errorf("Match() missing client_id %v", wantID)
				}
			}
		})
	}
}

func TestMatcher_UpdateIndexes(t *testing.T) {
	snap1 := &snapshot.Snapshot{
		BySeverity: map[string][]int{"HIGH": {1}},
		BySource:   map[string][]int{"service-a": {1}},
		ByName:     map[string][]int{"disk-full": {1}},
		Rules:      map[int]snapshot.RuleInfo{1: {RuleID: "rule-1", ClientID: "client-1"}},
	}
	idx1 := indexes.NewIndexes(snap1)
	matcher := NewMatcher(idx1)

	// Verify initial state
	if matcher.RuleCount() != 1 {
		t.Errorf("Initial RuleCount() = %v, want 1", matcher.RuleCount())
	}

	// Update with new indexes
	snap2 := &snapshot.Snapshot{
		BySeverity: map[string][]int{"HIGH": {1, 2}, "LOW": {3}},
		BySource:   map[string][]int{"service-a": {1}, "service-b": {2, 3}},
		ByName:     map[string][]int{"disk-full": {1}, "cpu-high": {2, 3}},
		Rules: map[int]snapshot.RuleInfo{
			1: {RuleID: "rule-1", ClientID: "client-1"},
			2: {RuleID: "rule-2", ClientID: "client-1"},
			3: {RuleID: "rule-3", ClientID: "client-2"},
		},
	}
	idx2 := indexes.NewIndexes(snap2)
	matcher.UpdateIndexes(idx2)

	// Verify updated state
	if matcher.RuleCount() != 3 {
		t.Errorf("Updated RuleCount() = %v, want 3", matcher.RuleCount())
	}

	// Verify new rules are matched
	result := matcher.Match("LOW", "service-b", "cpu-high")
	if len(result) != 1 {
		t.Fatalf("Match() after update returned %d clients, want 1", len(result))
	}
	if _, exists := result["client-2"]; !exists {
		t.Error("Match() after update missing client-2")
	}
}

func TestMatcher_ConcurrentAccess(t *testing.T) {
	snap := &snapshot.Snapshot{
		BySeverity: map[string][]int{"HIGH": {1}},
		BySource:   map[string][]int{"service-a": {1}},
		ByName:     map[string][]int{"disk-full": {1}},
		Rules:      map[int]snapshot.RuleInfo{1: {RuleID: "rule-1", ClientID: "client-1"}},
	}
	idx := indexes.NewIndexes(snap)
	matcher := NewMatcher(idx)

	// Test concurrent reads
	var wg sync.WaitGroup
	numGoroutines := 10
	numReads := 100

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numReads; j++ {
				_ = matcher.Match("HIGH", "service-a", "disk-full")
				_ = matcher.RuleCount()
			}
		}()
	}
	wg.Wait()

	// Test concurrent read and write
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < numReads; i++ {
			_ = matcher.Match("HIGH", "service-a", "disk-full")
		}
	}()
	go func() {
		defer wg.Done()
		newSnap := &snapshot.Snapshot{
			BySeverity: map[string][]int{"LOW": {2}},
			BySource:   map[string][]int{"service-b": {2}},
			ByName:     map[string][]int{"cpu-high": {2}},
			Rules:      map[int]snapshot.RuleInfo{2: {RuleID: "rule-2", ClientID: "client-2"}},
		}
		newIdx := indexes.NewIndexes(newSnap)
		matcher.UpdateIndexes(newIdx)
	}()
	wg.Wait()

	// Should not panic and should have valid state
	if matcher.RuleCount() < 0 {
		t.Error("Matcher in invalid state after concurrent access")
	}
}

func TestMatcher_RuleCount(t *testing.T) {
	tests := []struct {
		name  string
		snap  *snapshot.Snapshot
		want  int
	}{
		{
			name: "empty",
			snap: &snapshot.Snapshot{
				Rules: map[int]snapshot.RuleInfo{},
			},
			want: 0,
		},
		{
			name: "single rule",
			snap: &snapshot.Snapshot{
				BySeverity: map[string][]int{"HIGH": {1}},
				BySource:   map[string][]int{"service-a": {1}},
				ByName:     map[string][]int{"disk-full": {1}},
				Rules:      map[int]snapshot.RuleInfo{1: {RuleID: "rule-1", ClientID: "client-1"}},
			},
			want: 1,
		},
		{
			name: "multiple rules",
			snap: &snapshot.Snapshot{
				BySeverity: map[string][]int{"HIGH": {1, 2}},
				BySource:   map[string][]int{"service-a": {1, 2}},
				ByName:     map[string][]int{"disk-full": {1, 2}},
				Rules: map[int]snapshot.RuleInfo{
					1: {RuleID: "rule-1", ClientID: "client-1"},
					2: {RuleID: "rule-2", ClientID: "client-2"},
				},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := indexes.NewIndexes(tt.snap)
			matcher := NewMatcher(idx)
			if got := matcher.RuleCount(); got != tt.want {
				t.Errorf("RuleCount() = %v, want %v", got, tt.want)
			}
		})
	}
}
