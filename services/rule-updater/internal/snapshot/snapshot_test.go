package snapshot

import (
	"context"
	"encoding/json"
	"testing"

	"rule-updater/internal/database"
	"github.com/redis/go-redis/v9"
)

func TestNewWriter(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	writer := NewWriter(client)
	if writer == nil {
		t.Fatal("NewWriter() returned nil")
	}
	if writer.client != client {
		t.Error("NewWriter() client not set correctly")
	}
	if writer.addRuleScript == nil {
		t.Error("NewWriter() addRuleScript not initialized")
	}
	if writer.removeRuleScript == nil {
		t.Error("NewWriter() removeRuleScript not initialized")
	}
}

func TestBuildSnapshot(t *testing.T) {
	tests := []struct {
		name  string
		rules []*database.Rule
		want  int // expected number of rules in snapshot
	}{
		{
			name:  "empty rules",
			rules: []*database.Rule{},
			want:  0,
		},
		{
			name: "single rule",
			rules: []*database.Rule{
				{
					RuleID:   "rule-1",
					ClientID: "client-1",
					Severity: "HIGH",
					Source:   "service-a",
					Name:     "disk-full",
					Enabled:  true,
				},
			},
			want: 1,
		},
		{
			name: "multiple rules",
			rules: []*database.Rule{
				{
					RuleID:   "rule-1",
					ClientID: "client-1",
					Severity: "HIGH",
					Source:   "service-a",
					Name:     "disk-full",
					Enabled:  true,
				},
				{
					RuleID:   "rule-2",
					ClientID: "client-2",
					Severity: "MEDIUM",
					Source:   "service-b",
					Name:     "cpu-high",
					Enabled:  true,
				},
			},
			want: 2,
		},
		{
			name: "rules with same severity",
			rules: []*database.Rule{
				{
					RuleID:   "rule-1",
					ClientID: "client-1",
					Severity: "HIGH",
					Source:   "service-a",
					Name:     "disk-full",
					Enabled:  true,
				},
				{
					RuleID:   "rule-2",
					ClientID: "client-2",
					Severity: "HIGH",
					Source:   "service-b",
					Name:     "cpu-high",
					Enabled:  true,
				},
			},
			want: 2,
		},
		{
			name: "rules with wildcards",
			rules: []*database.Rule{
				{
					RuleID:   "rule-1",
					ClientID: "client-1",
					Severity: "*",
					Source:   "service-a",
					Name:     "disk-full",
					Enabled:  true,
				},
				{
					RuleID:   "rule-2",
					ClientID: "client-2",
					Severity: "HIGH",
					Source:   "*",
					Name:     "cpu-high",
					Enabled:  true,
				},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snap := BuildSnapshot(tt.rules)
			if snap == nil {
				t.Fatal("BuildSnapshot() returned nil")
			}
			if snap.SchemaVersion != SchemaVersion {
				t.Errorf("BuildSnapshot() SchemaVersion = %v, want %v", snap.SchemaVersion, SchemaVersion)
			}
			if len(snap.Rules) != tt.want {
				t.Errorf("BuildSnapshot() Rules count = %v, want %v", len(snap.Rules), tt.want)
			}

			// Verify dictionaries are built
			for _, rule := range tt.rules {
				if _, exists := snap.SeverityDict[rule.Severity]; !exists && len(tt.rules) > 0 {
					t.Errorf("BuildSnapshot() SeverityDict missing %v", rule.Severity)
				}
				if _, exists := snap.SourceDict[rule.Source]; !exists && len(tt.rules) > 0 {
					t.Errorf("BuildSnapshot() SourceDict missing %v", rule.Source)
				}
				if _, exists := snap.NameDict[rule.Name]; !exists && len(tt.rules) > 0 {
					t.Errorf("BuildSnapshot() NameDict missing %v", rule.Name)
				}
			}

			// Verify indexes are built
			for _, rule := range tt.rules {
				if ruleInts, exists := snap.BySeverity[rule.Severity]; exists {
					found := false
					for _, ruleInt := range ruleInts {
						if ruleInfo, ok := snap.Rules[ruleInt]; ok && ruleInfo.RuleID == rule.RuleID {
							found = true
							break
						}
					}
					if !found && len(tt.rules) > 0 {
						t.Errorf("BuildSnapshot() rule %v not found in BySeverity index", rule.RuleID)
					}
				}
			}
		})
	}
}

func TestSnapshot_findRuleInt(t *testing.T) {
	snap := &Snapshot{
		Rules: map[int]RuleInfo{
			1: {RuleID: "rule-1", ClientID: "client-1"},
			2: {RuleID: "rule-2", ClientID: "client-2"},
		},
	}

	tests := []struct {
		name   string
		ruleID string
		want   int
	}{
		{
			name:   "existing rule",
			ruleID: "rule-1",
			want:   1,
		},
		{
			name:   "non-existing rule",
			ruleID: "rule-999",
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snap.findRuleInt(tt.ruleID)
			if got != tt.want {
				t.Errorf("findRuleInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSnapshot_getNextRuleInt(t *testing.T) {
	tests := []struct {
		name string
		snap *Snapshot
		want int
	}{
		{
			name: "empty snapshot",
			snap: &Snapshot{
				Rules: map[int]RuleInfo{},
			},
			want: 1,
		},
		{
			name: "snapshot with rules",
			snap: &Snapshot{
				Rules: map[int]RuleInfo{
					1: {RuleID: "rule-1", ClientID: "client-1"},
					2: {RuleID: "rule-2", ClientID: "client-2"},
					5: {RuleID: "rule-5", ClientID: "client-5"},
				},
			},
			want: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.snap.getNextRuleInt()
			if got != tt.want {
				t.Errorf("getNextRuleInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemoveFromSlice(t *testing.T) {
	tests := []struct {
		name  string
		slice []int
		value int
		want  []int
	}{
		{
			name:  "remove from middle",
			slice: []int{1, 2, 3, 4, 5},
			value: 3,
			want:  []int{1, 2, 4, 5},
		},
		{
			name:  "remove from beginning",
			slice: []int{1, 2, 3},
			value: 1,
			want:  []int{2, 3},
		},
		{
			name:  "remove from end",
			slice: []int{1, 2, 3},
			value: 3,
			want:  []int{1, 2},
		},
		{
			name:  "remove non-existing",
			slice: []int{1, 2, 3},
			value: 999,
			want:  []int{1, 2, 3},
		},
		{
			name:  "remove from empty",
			slice: []int{},
			value: 1,
			want:  []int{},
		},
		{
			name:  "remove duplicate",
			slice: []int{1, 2, 2, 3},
			value: 2,
			want:  []int{1, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeFromSlice(tt.slice, tt.value)
			if len(got) != len(tt.want) {
				t.Errorf("removeFromSlice() len = %v, want %v", len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("removeFromSlice() [%d] = %v, want %v", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestSnapshot_AddRule(t *testing.T) {
	tests := []struct {
		name    string
		initial *Snapshot
		rule    *database.Rule
		wantErr bool
		wantLen int
	}{
		{
			name: "add new rule",
			initial: &Snapshot{
				SchemaVersion: SchemaVersion,
				SeverityDict:  make(map[string]int),
				SourceDict:    make(map[string]int),
				NameDict:      make(map[string]int),
				BySeverity:    make(map[string][]int),
				BySource:      make(map[string][]int),
				ByName:        make(map[string][]int),
				Rules:         make(map[int]RuleInfo),
			},
			rule: &database.Rule{
				RuleID:   "rule-1",
				ClientID: "client-1",
				Severity: "HIGH",
				Source:   "service-a",
				Name:     "disk-full",
				Enabled:  true,
			},
			wantErr: false,
			wantLen: 1,
		},
		{
			name: "add disabled rule (should not add)",
			initial: &Snapshot{
				SchemaVersion: SchemaVersion,
				SeverityDict:  make(map[string]int),
				SourceDict:    make(map[string]int),
				NameDict:      make(map[string]int),
				BySeverity:    make(map[string][]int),
				BySource:      make(map[string][]int),
				ByName:        make(map[string][]int),
				Rules:         make(map[int]RuleInfo),
			},
			rule: &database.Rule{
				RuleID:   "rule-1",
				ClientID: "client-1",
				Severity: "HIGH",
				Source:   "service-a",
				Name:     "disk-full",
				Enabled:  false,
			},
			wantErr: false,
			wantLen: 0,
		},
		{
			name: "add existing rule (should update)",
			initial: &Snapshot{
				SchemaVersion: SchemaVersion,
				SeverityDict:  map[string]int{"HIGH": 1},
				SourceDict:    map[string]int{"service-a": 1},
				NameDict:      map[string]int{"disk-full": 1},
				BySeverity:    map[string][]int{"HIGH": {1}},
				BySource:      map[string][]int{"service-a": {1}},
				ByName:        map[string][]int{"disk-full": {1}},
				Rules: map[int]RuleInfo{
					1: {RuleID: "rule-1", ClientID: "client-1"},
				},
			},
			rule: &database.Rule{
				RuleID:   "rule-1",
				ClientID: "client-1",
				Severity: "MEDIUM",
				Source:   "service-b",
				Name:     "cpu-high",
				Enabled:  true,
			},
			wantErr: false,
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.initial.AddRule(tt.rule)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddRule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(tt.initial.Rules) != tt.wantLen {
				t.Errorf("AddRule() Rules count = %v, want %v", len(tt.initial.Rules), tt.wantLen)
			}
		})
	}
}

func TestSnapshot_UpdateRule(t *testing.T) {
	tests := []struct {
		name    string
		initial *Snapshot
		rule    *database.Rule
		wantErr bool
		wantLen int
	}{
		{
			name: "update existing rule",
			initial: &Snapshot{
				SchemaVersion: SchemaVersion,
				SeverityDict:  map[string]int{"HIGH": 1},
				SourceDict:    map[string]int{"service-a": 1},
				NameDict:      map[string]int{"disk-full": 1},
				BySeverity:    map[string][]int{"HIGH": {1}},
				BySource:      map[string][]int{"service-a": {1}},
				ByName:        map[string][]int{"disk-full": {1}},
				Rules: map[int]RuleInfo{
					1: {RuleID: "rule-1", ClientID: "client-1"},
				},
			},
			rule: &database.Rule{
				RuleID:   "rule-1",
				ClientID: "client-1",
				Severity: "MEDIUM",
				Source:   "service-b",
				Name:     "cpu-high",
				Enabled:  true,
			},
			wantErr: false,
			wantLen: 1,
		},
		{
			name: "update non-existing rule (should add)",
			initial: &Snapshot{
				SchemaVersion: SchemaVersion,
				SeverityDict:  make(map[string]int),
				SourceDict:    make(map[string]int),
				NameDict:      make(map[string]int),
				BySeverity:    make(map[string][]int),
				BySource:      make(map[string][]int),
				ByName:        make(map[string][]int),
				Rules:         make(map[int]RuleInfo),
			},
			rule: &database.Rule{
				RuleID:   "rule-1",
				ClientID: "client-1",
				Severity: "HIGH",
				Source:   "service-a",
				Name:     "disk-full",
				Enabled:  true,
			},
			wantErr: false,
			wantLen: 1,
		},
		{
			name: "update to disabled (should remove)",
			initial: &Snapshot{
				SchemaVersion: SchemaVersion,
				SeverityDict:  map[string]int{"HIGH": 1},
				SourceDict:    map[string]int{"service-a": 1},
				NameDict:      map[string]int{"disk-full": 1},
				BySeverity:    map[string][]int{"HIGH": {1}},
				BySource:      map[string][]int{"service-a": {1}},
				ByName:        map[string][]int{"disk-full": {1}},
				Rules: map[int]RuleInfo{
					1: {RuleID: "rule-1", ClientID: "client-1"},
				},
			},
			rule: &database.Rule{
				RuleID:   "rule-1",
				ClientID: "client-1",
				Severity: "HIGH",
				Source:   "service-a",
				Name:     "disk-full",
				Enabled:  false,
			},
			wantErr: false,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.initial.UpdateRule(tt.rule)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateRule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(tt.initial.Rules) != tt.wantLen {
				t.Errorf("UpdateRule() Rules count = %v, want %v", len(tt.initial.Rules), tt.wantLen)
			}
		})
	}
}

func TestSnapshot_RemoveRule(t *testing.T) {
	tests := []struct {
		name    string
		initial *Snapshot
		ruleID  string
		wantErr bool
		wantLen int
	}{
		{
			name: "remove existing rule",
			initial: &Snapshot{
				SchemaVersion: SchemaVersion,
				SeverityDict:  map[string]int{"HIGH": 1},
				SourceDict:    map[string]int{"service-a": 1},
				NameDict:      map[string]int{"disk-full": 1},
				BySeverity:    map[string][]int{"HIGH": {1}},
				BySource:      map[string][]int{"service-a": {1}},
				ByName:        map[string][]int{"disk-full": {1}},
				Rules: map[int]RuleInfo{
					1: {RuleID: "rule-1", ClientID: "client-1"},
				},
			},
			ruleID:  "rule-1",
			wantErr: false,
			wantLen: 0,
		},
		{
			name: "remove non-existing rule",
			initial: &Snapshot{
				SchemaVersion: SchemaVersion,
				SeverityDict:  map[string]int{"HIGH": 1},
				SourceDict:    map[string]int{"service-a": 1},
				NameDict:      map[string]int{"disk-full": 1},
				BySeverity:    map[string][]int{"HIGH": {1}},
				BySource:      map[string][]int{"service-a": {1}},
				ByName:        map[string][]int{"disk-full": {1}},
				Rules: map[int]RuleInfo{
					1: {RuleID: "rule-1", ClientID: "client-1"},
				},
			},
			ruleID:  "rule-999",
			wantErr: false,
			wantLen: 1,
		},
		{
			name: "remove from empty snapshot",
			initial: &Snapshot{
				SchemaVersion: SchemaVersion,
				SeverityDict:  make(map[string]int),
				SourceDict:    make(map[string]int),
				NameDict:      make(map[string]int),
				BySeverity:    make(map[string][]int),
				BySource:      make(map[string][]int),
				ByName:        make(map[string][]int),
				Rules:         make(map[int]RuleInfo),
			},
			ruleID:  "rule-1",
			wantErr: false,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.initial.RemoveRule(tt.ruleID)
			if (err != nil) != tt.wantErr {
				t.Errorf("RemoveRule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(tt.initial.Rules) != tt.wantLen {
				t.Errorf("RemoveRule() Rules count = %v, want %v", len(tt.initial.Rules), tt.wantLen)
			}
		})
	}
}

func TestWriter_WriteSnapshot_Integration(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Skipping integration test: Redis not available: %v", err)
	}

	writer := NewWriter(client)

	// Clean up before test
	client.Del(ctx, SnapshotKey, VersionKey)

	snap := &Snapshot{
		SchemaVersion: SchemaVersion,
		SeverityDict:  map[string]int{"HIGH": 1, "MEDIUM": 2},
		SourceDict:    map[string]int{"service-a": 1},
		NameDict:      map[string]int{"disk-full": 1},
		BySeverity:    map[string][]int{"HIGH": {1}},
		BySource:      map[string][]int{"service-a": {1}},
		ByName:        map[string][]int{"disk-full": {1}},
		Rules: map[int]RuleInfo{
			1: {RuleID: "rule-1", ClientID: "client-1"},
		},
	}

	if err := writer.WriteSnapshot(ctx, snap); err != nil {
		t.Fatalf("WriteSnapshot() error = %v, want nil", err)
	}

	// Verify snapshot was written
	data, err := client.Get(ctx, SnapshotKey).Bytes()
	if err != nil {
		t.Fatalf("Failed to get snapshot from Redis: %v", err)
	}

	var loadedSnap Snapshot
	if err := json.Unmarshal(data, &loadedSnap); err != nil {
		t.Fatalf("Failed to unmarshal snapshot: %v", err)
	}

	if loadedSnap.SchemaVersion != snap.SchemaVersion {
		t.Errorf("WriteSnapshot() SchemaVersion = %v, want %v", loadedSnap.SchemaVersion, snap.SchemaVersion)
	}
	if len(loadedSnap.Rules) != len(snap.Rules) {
		t.Errorf("WriteSnapshot() Rules count = %v, want %v", len(loadedSnap.Rules), len(snap.Rules))
	}

	// Verify version was incremented
	version, err := writer.GetVersion(ctx)
	if err != nil {
		t.Fatalf("GetVersion() error = %v, want nil", err)
	}
	if version <= 0 {
		t.Errorf("GetVersion() = %v, want > 0", version)
	}

	// Clean up
	client.Del(ctx, SnapshotKey, VersionKey)
}

func TestWriter_GetVersion_Integration(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Skipping integration test: Redis not available: %v", err)
	}

	writer := NewWriter(client)

	// Clean up before test
	client.Del(ctx, VersionKey)

	// Test GetVersion when version doesn't exist (should return 0)
	version, err := writer.GetVersion(ctx)
	if err != nil {
		t.Fatalf("GetVersion() error = %v, want nil", err)
	}
	if version != 0 {
		t.Errorf("GetVersion() = %v, want 0", version)
	}

	// Set version and test
	client.Set(ctx, VersionKey, 42, 0)
	version, err = writer.GetVersion(ctx)
	if err != nil {
		t.Fatalf("GetVersion() error = %v, want nil", err)
	}
	if version != 42 {
		t.Errorf("GetVersion() = %v, want 42", version)
	}

	// Clean up
	client.Del(ctx, VersionKey)
}

func TestWriter_LoadSnapshot_Integration(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Skipping integration test: Redis not available: %v", err)
	}

	writer := NewWriter(client)

	// Clean up before test
	client.Del(ctx, SnapshotKey)

	// Test LoadSnapshot when snapshot doesn't exist (should return empty snapshot)
	snap, err := writer.LoadSnapshot(ctx)
	if err != nil {
		t.Fatalf("LoadSnapshot() error = %v, want nil", err)
	}
	if snap == nil {
		t.Fatal("LoadSnapshot() returned nil")
	}
	if snap.SchemaVersion != SchemaVersion {
		t.Errorf("LoadSnapshot() SchemaVersion = %v, want %v", snap.SchemaVersion, SchemaVersion)
	}
	if len(snap.Rules) != 0 {
		t.Errorf("LoadSnapshot() Rules count = %v, want 0", len(snap.Rules))
	}

	// Write a snapshot and test loading it
	testSnap := &Snapshot{
		SchemaVersion: SchemaVersion,
		SeverityDict:  map[string]int{"HIGH": 1},
		SourceDict:    map[string]int{"service-a": 1},
		NameDict:      map[string]int{"disk-full": 1},
		BySeverity:    map[string][]int{"HIGH": {1}},
		BySource:      map[string][]int{"service-a": {1}},
		ByName:        map[string][]int{"disk-full": {1}},
		Rules: map[int]RuleInfo{
			1: {RuleID: "rule-1", ClientID: "client-1"},
		},
	}

	if err := writer.WriteSnapshot(ctx, testSnap); err != nil {
		t.Fatalf("WriteSnapshot() error = %v, want nil", err)
	}

	loadedSnap, err := writer.LoadSnapshot(ctx)
	if err != nil {
		t.Fatalf("LoadSnapshot() error = %v, want nil", err)
	}
	if loadedSnap.SchemaVersion != testSnap.SchemaVersion {
		t.Errorf("LoadSnapshot() SchemaVersion = %v, want %v", loadedSnap.SchemaVersion, testSnap.SchemaVersion)
	}
	if len(loadedSnap.Rules) != len(testSnap.Rules) {
		t.Errorf("LoadSnapshot() Rules count = %v, want %v", len(loadedSnap.Rules), len(testSnap.Rules))
	}

	// Clean up
	client.Del(ctx, SnapshotKey, VersionKey)
}

func TestWriter_AddRuleDirect_Integration(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Skipping integration test: Redis not available: %v", err)
	}

	writer := NewWriter(client)

	// Clean up before test
	client.Del(ctx, SnapshotKey, VersionKey)

	rule := &database.Rule{
		RuleID:   "rule-1",
		ClientID: "client-1",
		Severity: "HIGH",
		Source:   "service-a",
		Name:     "disk-full",
		Enabled:  true,
	}

	if err := writer.AddRuleDirect(ctx, rule); err != nil {
		t.Fatalf("AddRuleDirect() error = %v, want nil", err)
	}

	// Verify rule was added
	snap, err := writer.LoadSnapshot(ctx)
	if err != nil {
		t.Fatalf("LoadSnapshot() error = %v, want nil", err)
	}

	found := false
	for _, ruleInfo := range snap.Rules {
		if ruleInfo.RuleID == rule.RuleID {
			found = true
			break
		}
	}
	if !found {
		t.Error("AddRuleDirect() rule not found in snapshot")
	}

	// Test adding disabled rule (should not add)
	disabledRule := &database.Rule{
		RuleID:   "rule-2",
		ClientID: "client-2",
		Severity: "MEDIUM",
		Source:   "service-b",
		Name:     "cpu-high",
		Enabled:  false,
	}

	if err := writer.AddRuleDirect(ctx, disabledRule); err != nil {
		t.Fatalf("AddRuleDirect() error = %v, want nil", err)
	}

	// Verify disabled rule was not added
	snap, err = writer.LoadSnapshot(ctx)
	if err != nil {
		t.Fatalf("LoadSnapshot() error = %v, want nil", err)
	}

	found = false
	for _, ruleInfo := range snap.Rules {
		if ruleInfo.RuleID == disabledRule.RuleID {
			found = true
			break
		}
	}
	if found {
		t.Error("AddRuleDirect() disabled rule should not be added")
	}

	// Clean up
	client.Del(ctx, SnapshotKey, VersionKey)
}

func TestWriter_RemoveRuleDirect_Integration(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Skipping integration test: Redis not available: %v", err)
	}

	writer := NewWriter(client)

	// Clean up before test
	client.Del(ctx, SnapshotKey, VersionKey)

	// Add a rule first
	rule := &database.Rule{
		RuleID:   "rule-1",
		ClientID: "client-1",
		Severity: "HIGH",
		Source:   "service-a",
		Name:     "disk-full",
		Enabled:  true,
	}

	if err := writer.AddRuleDirect(ctx, rule); err != nil {
		t.Fatalf("AddRuleDirect() error = %v, want nil", err)
	}

	// Remove the rule
	if err := writer.RemoveRuleDirect(ctx, rule.RuleID); err != nil {
		t.Fatalf("RemoveRuleDirect() error = %v, want nil", err)
	}

	// Verify rule was removed
	snap, err := writer.LoadSnapshot(ctx)
	if err != nil {
		t.Fatalf("LoadSnapshot() error = %v, want nil", err)
	}

	found := false
	for _, ruleInfo := range snap.Rules {
		if ruleInfo.RuleID == rule.RuleID {
			found = true
			break
		}
	}
	if found {
		t.Error("RemoveRuleDirect() rule still found in snapshot")
	}

	// Test removing non-existing rule (should not error)
	if err := writer.RemoveRuleDirect(ctx, "rule-999"); err != nil {
		t.Fatalf("RemoveRuleDirect() error = %v, want nil", err)
	}

	// Clean up
	client.Del(ctx, SnapshotKey, VersionKey)
}

func TestWriter_LoadSnapshot_InvalidJSON(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Skipping integration test: Redis not available: %v", err)
	}

	writer := NewWriter(client)

	// Set invalid JSON
	client.Set(ctx, SnapshotKey, "invalid json", 0)

	_, err := writer.LoadSnapshot(ctx)
	if err == nil {
		t.Error("LoadSnapshot() expected error for invalid JSON, got nil")
	}

	// Clean up
	client.Del(ctx, SnapshotKey)
}

func TestSnapshot_JSONRoundTrip(t *testing.T) {
	snap := &Snapshot{
		SchemaVersion: SchemaVersion,
		SeverityDict:  map[string]int{"HIGH": 1, "MEDIUM": 2},
		SourceDict:    map[string]int{"service-a": 1, "service-b": 2},
		NameDict:      map[string]int{"disk-full": 1, "cpu-high": 2},
		BySeverity:    map[string][]int{"HIGH": {1, 2}, "MEDIUM": {3}},
		BySource:      map[string][]int{"service-a": {1}, "service-b": {2, 3}},
		ByName:        map[string][]int{"disk-full": {1}, "cpu-high": {2, 3}},
		Rules: map[int]RuleInfo{
			1: {RuleID: "rule-1", ClientID: "client-1"},
			2: {RuleID: "rule-2", ClientID: "client-1"},
			3: {RuleID: "rule-3", ClientID: "client-2"},
		},
	}

	data, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("Failed to marshal snapshot: %v", err)
	}

	var unmarshaled Snapshot
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal snapshot: %v", err)
	}

	if unmarshaled.SchemaVersion != snap.SchemaVersion {
		t.Errorf("SchemaVersion = %v, want %v", unmarshaled.SchemaVersion, snap.SchemaVersion)
	}
	if len(unmarshaled.Rules) != len(snap.Rules) {
		t.Errorf("Rules count = %v, want %v", len(unmarshaled.Rules), len(snap.Rules))
	}
}

func TestRuleInfo_JSONRoundTrip(t *testing.T) {
	ruleInfo := RuleInfo{
		RuleID:   "rule-123",
		ClientID: "client-456",
	}

	data, err := json.Marshal(ruleInfo)
	if err != nil {
		t.Fatalf("Failed to marshal RuleInfo: %v", err)
	}

	var unmarshaled RuleInfo
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal RuleInfo: %v", err)
	}

	if unmarshaled.RuleID != ruleInfo.RuleID {
		t.Errorf("RuleID = %v, want %v", unmarshaled.RuleID, ruleInfo.RuleID)
	}
	if unmarshaled.ClientID != ruleInfo.ClientID {
		t.Errorf("ClientID = %v, want %v", unmarshaled.ClientID, ruleInfo.ClientID)
	}
}
