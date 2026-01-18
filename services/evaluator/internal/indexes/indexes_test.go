package indexes

import (
	"evaluator/internal/snapshot"
	"reflect"
	"testing"
)

func TestNewIndexes(t *testing.T) {
	snap := &snapshot.Snapshot{
		SchemaVersion: 1,
		BySeverity:    map[string][]int{"HIGH": {1, 2}, "LOW": {3}},
		BySource:      map[string][]int{"service-a": {1}, "service-b": {2, 3}},
		ByName:        map[string][]int{"disk-full": {1}, "cpu-high": {2, 3}},
		Rules: map[int]snapshot.RuleInfo{
			1: {RuleID: "rule-1", ClientID: "client-1"},
			2: {RuleID: "rule-2", ClientID: "client-1"},
			3: {RuleID: "rule-3", ClientID: "client-2"},
		},
	}

	idx := NewIndexes(snap)
	if idx == nil {
		t.Fatal("NewIndexes() returned nil")
	}

	// Verify deep copy - modifying original shouldn't affect indexes
	snap.BySeverity["HIGH"] = append(snap.BySeverity["HIGH"], 999)
	if len(idx.bySeverity["HIGH"]) == len(snap.BySeverity["HIGH"]) {
		t.Error("NewIndexes() did not create deep copy of BySeverity")
	}

	// Verify rule count
	if idx.RuleCount() != 3 {
		t.Errorf("RuleCount() = %v, want 3", idx.RuleCount())
	}
}

func TestIndexes_Match(t *testing.T) {
	snap := &snapshot.Snapshot{
		SchemaVersion: 1,
		BySeverity: map[string][]int{
			"HIGH":   {1, 2},
			"LOW":    {3},
			"MEDIUM": {4},
			"*":      {5}, // wildcard rule
		},
		BySource: map[string][]int{
			"service-a": {1, 3},
			"service-b": {2},
			"*":         {4, 5}, // wildcard rules
		},
		ByName: map[string][]int{
			"disk-full": {1, 2},
			"cpu-high":  {3},
			"*":         {4, 5}, // wildcard rules
		},
		Rules: map[int]snapshot.RuleInfo{
			1: {RuleID: "rule-1", ClientID: "client-1"},
			2: {RuleID: "rule-2", ClientID: "client-1"},
			3: {RuleID: "rule-3", ClientID: "client-2"},
			4: {RuleID: "rule-4", ClientID: "client-2"},
			5: {RuleID: "rule-5", ClientID: "client-3"},
		},
	}

	idx := NewIndexes(snap)

	tests := []struct {
		name           string
		severity       string
		source         string
		nameField      string
		wantClientIDs  []string
		wantRuleCounts map[string]int // client_id -> expected rule count
	}{
		{
			name:          "exact match - single rule",
			severity:      "HIGH",
			source:        "service-a",
			nameField:     "disk-full",
			wantClientIDs: []string{"client-1", "client-3"}, // rule-1 (exact) + rule-5 (wildcard matches all)
			wantRuleCounts: map[string]int{"client-1": 1, "client-3": 1},
		},
		{
			name:          "exact match - multiple rules same client",
			severity:      "HIGH",
			source:        "service-b",
			nameField:     "disk-full",
			wantClientIDs: []string{"client-1", "client-3"}, // rule-2 (exact) + rule-5 (wildcard)
			wantRuleCounts: map[string]int{"client-1": 1, "client-3": 1},
		},
		{
			name:          "exact match - different client",
			severity:      "LOW",
			source:        "service-a",
			nameField:     "cpu-high",
			wantClientIDs: []string{"client-2", "client-3"}, // rule-3 (exact) + rule-5 (wildcard)
			wantRuleCounts: map[string]int{"client-2": 1, "client-3": 1},
		},
		{
			name:          "wildcard severity match",
			severity:      "CRITICAL", // not in index, but wildcard should match
			source:        "service-a",
			nameField:     "disk-full",
			wantClientIDs: []string{"client-3"}, // rule-5 (all wildcards) - rule-1 requires HIGH severity, not CRITICAL
			wantRuleCounts: map[string]int{"client-3": 1},
		},
		{
			name:          "wildcard source match",
			severity:      "MEDIUM",
			source:        "service-c", // not in index
			nameField:     "disk-full",
			wantClientIDs: []string{"client-2", "client-3"},
			wantRuleCounts: map[string]int{"client-2": 1, "client-3": 1}, // rule-4 (wildcard source) + rule-5 (wildcard)
		},
		{
			name:          "wildcard name match",
			severity:      "MEDIUM",
			source:        "service-a",
			nameField:     "memory-high", // not in index
			wantClientIDs: []string{"client-2", "client-3"},
			wantRuleCounts: map[string]int{"client-2": 1, "client-3": 1}, // rule-4 (wildcard name) + rule-5 (wildcard)
		},
		{
			name:          "all wildcards match",
			severity:      "UNKNOWN",
			source:        "unknown-service",
			nameField:     "unknown-alert",
			wantClientIDs: []string{"client-3"},
			wantRuleCounts: map[string]int{"client-3": 1}, // rule-5 (all wildcards)
		},
		{
			name:          "no match",
			severity:      "LOW",
			source:        "service-c",
			nameField:     "unknown-alert",
			wantClientIDs: []string{"client-3"}, // rule-5 (all wildcards matches everything)
			wantRuleCounts: map[string]int{"client-3": 1},
		},
		{
			name:          "empty result when no candidates",
			severity:      "NONE",
			source:        "NONE",
			nameField:     "NONE",
			wantClientIDs: []string{"client-3"}, // rule-5 (all wildcards matches everything)
			wantRuleCounts: map[string]int{"client-3": 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := idx.Match(tt.severity, tt.source, tt.nameField)

			// Check client IDs
			gotClientIDs := make([]string, 0, len(result))
			for clientID := range result {
				gotClientIDs = append(gotClientIDs, clientID)
			}

			if len(gotClientIDs) != len(tt.wantClientIDs) {
				t.Errorf("Match() returned %d clients, want %d", len(gotClientIDs), len(tt.wantClientIDs))
			}

			// Check rule counts per client
			for clientID, wantCount := range tt.wantRuleCounts {
				gotRules, exists := result[clientID]
				if !exists {
					t.Errorf("Match() missing client_id %v", clientID)
					continue
				}
				if len(gotRules) != wantCount {
					t.Errorf("Match() client %v has %d rules, want %d", clientID, len(gotRules), wantCount)
				}
			}

			// Verify no extra clients
			for clientID := range result {
				found := false
				for _, wantID := range tt.wantClientIDs {
					if clientID == wantID {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Match() returned unexpected client_id %v", clientID)
				}
			}
		})
	}
}

func TestIndexes_Match_Intersection(t *testing.T) {
	// Test intersection logic with specific rule combinations
	snap := &snapshot.Snapshot{
		SchemaVersion: 1,
		BySeverity: map[string][]int{
			"HIGH": {1, 2, 3},
		},
		BySource: map[string][]int{
			"service-a": {1, 2},
			"service-b": {3},
		},
		ByName: map[string][]int{
			"disk-full": {1},
			"cpu-high":  {2, 3},
		},
		Rules: map[int]snapshot.RuleInfo{
			1: {RuleID: "rule-1", ClientID: "client-1"}, // HIGH + service-a + disk-full
			2: {RuleID: "rule-2", ClientID: "client-1"}, // HIGH + service-a + cpu-high
			3: {RuleID: "rule-3", ClientID: "client-2"}, // HIGH + service-b + cpu-high
		},
	}

	idx := NewIndexes(snap)

	// Test: HIGH + service-a + disk-full should match only rule-1
	result := idx.Match("HIGH", "service-a", "disk-full")
	if len(result) != 1 {
		t.Fatalf("Match() returned %d clients, want 1", len(result))
	}
	rules, exists := result["client-1"]
	if !exists {
		t.Fatal("Match() missing client-1")
	}
	if len(rules) != 1 || rules[0] != "rule-1" {
		t.Errorf("Match() rules = %v, want [rule-1]", rules)
	}

	// Test: HIGH + service-a + cpu-high should match rule-2
	result = idx.Match("HIGH", "service-a", "cpu-high")
	if len(result) != 1 {
		t.Fatalf("Match() returned %d clients, want 1", len(result))
	}
	rules, exists = result["client-1"]
	if !exists {
		t.Fatal("Match() missing client-1")
	}
	if len(rules) != 1 || rules[0] != "rule-2" {
		t.Errorf("Match() rules = %v, want [rule-2]", rules)
	}

	// Test: HIGH + service-b + cpu-high should match rule-3
	result = idx.Match("HIGH", "service-b", "cpu-high")
	if len(result) != 1 {
		t.Fatalf("Match() returned %d clients, want 1", len(result))
	}
	rules, exists = result["client-2"]
	if !exists {
		t.Fatal("Match() missing client-2")
	}
	if len(rules) != 1 || rules[0] != "rule-3" {
		t.Errorf("Match() rules = %v, want [rule-3]", rules)
	}
}

func TestIndexes_RuleCount(t *testing.T) {
	tests := []struct {
		name  string
		snap  *snapshot.Snapshot
		want  int
	}{
		{
			name: "empty snapshot",
			snap: &snapshot.Snapshot{
				Rules: map[int]snapshot.RuleInfo{},
			},
			want: 0,
		},
		{
			name: "single rule",
			snap: &snapshot.Snapshot{
				Rules: map[int]snapshot.RuleInfo{
					1: {RuleID: "rule-1", ClientID: "client-1"},
				},
			},
			want: 1,
		},
		{
			name: "multiple rules",
			snap: &snapshot.Snapshot{
				Rules: map[int]snapshot.RuleInfo{
					1: {RuleID: "rule-1", ClientID: "client-1"},
					2: {RuleID: "rule-2", ClientID: "client-1"},
					3: {RuleID: "rule-3", ClientID: "client-2"},
				},
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := NewIndexes(tt.snap)
			if got := idx.RuleCount(); got != tt.want {
				t.Errorf("RuleCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCombineLists(t *testing.T) {
	tests := []struct {
		name  string
		list1 []int
		list2 []int
		want  []int
	}{
		{
			name:  "both empty",
			list1: []int{},
			list2: []int{},
			want:  []int{},
		},
		{
			name:  "list1 empty",
			list1: []int{},
			list2: []int{1, 2, 3},
			want:  []int{1, 2, 3},
		},
		{
			name:  "list2 empty",
			list1: []int{1, 2, 3},
			list2: []int{},
			want:  []int{1, 2, 3},
		},
		{
			name:  "no duplicates",
			list1: []int{1, 2},
			list2: []int{3, 4},
			want:  []int{1, 2, 3, 4},
		},
		{
			name:  "with duplicates",
			list1: []int{1, 2, 3},
			list2: []int{2, 3, 4},
			want:  []int{1, 2, 3, 4},
		},
		{
			name:  "all duplicates",
			list1: []int{1, 2, 3},
			list2: []int{1, 2, 3},
			want:  []int{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := combineLists(tt.list1, tt.list2)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("combineLists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIndexes_Match_InvalidRuleInt(t *testing.T) {
	// Test that invalid ruleInt values are skipped
	snap := &snapshot.Snapshot{
		SchemaVersion: 1,
		BySeverity:    map[string][]int{"HIGH": {1, 999}}, // 999 doesn't exist in Rules
		BySource:      map[string][]int{"service-a": {1, 999}},
		ByName:        map[string][]int{"disk-full": {1, 999}},
		Rules: map[int]snapshot.RuleInfo{
			1: {RuleID: "rule-1", ClientID: "client-1"},
			// 999 is missing
		},
	}

	idx := NewIndexes(snap)
	result := idx.Match("HIGH", "service-a", "disk-full")

	// Should only return rule-1, not crash on 999
	if len(result) != 1 {
		t.Fatalf("Match() returned %d clients, want 1", len(result))
	}
	rules, exists := result["client-1"]
	if !exists {
		t.Fatal("Match() missing client-1")
	}
	if len(rules) != 1 || rules[0] != "rule-1" {
		t.Errorf("Match() rules = %v, want [rule-1]", rules)
	}
}
