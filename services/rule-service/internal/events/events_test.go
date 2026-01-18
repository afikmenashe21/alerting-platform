// Package events provides tests for event structures.
package events

import (
	"encoding/json"
	"testing"
)

// TestRuleChanged_JSONMarshaling tests JSON marshaling and unmarshaling of RuleChanged.
func TestRuleChanged_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name  string
		event RuleChanged
	}{
		{
			name: "CREATED action",
			event: RuleChanged{
				RuleID:        "rule-123",
				ClientID:       "client-456",
				Action:        ActionCreated,
				Version:        1,
				UpdatedAt:     1234567890,
				SchemaVersion:  1,
			},
		},
		{
			name: "UPDATED action",
			event: RuleChanged{
				RuleID:        "rule-123",
				ClientID:       "client-456",
				Action:        ActionUpdated,
				Version:        2,
				UpdatedAt:     1234567890,
				SchemaVersion:  1,
			},
		},
		{
			name: "DELETED action",
			event: RuleChanged{
				RuleID:        "rule-123",
				ClientID:       "client-456",
				Action:        ActionDeleted,
				Version:        5,
				UpdatedAt:     1234567890,
				SchemaVersion:  1,
			},
		},
		{
			name: "DISABLED action",
			event: RuleChanged{
				RuleID:        "rule-123",
				ClientID:       "client-456",
				Action:        ActionDisabled,
				Version:        3,
				UpdatedAt:     1234567890,
				SchemaVersion:  1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.event)
			if err != nil {
				t.Fatalf("Failed to marshal RuleChanged: %v", err)
			}

			// Test unmarshaling
			var unmarshaled RuleChanged
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal RuleChanged: %v", err)
			}

			// Verify all fields
			if unmarshaled.RuleID != tt.event.RuleID {
				t.Errorf("RuleID = %v, want %v", unmarshaled.RuleID, tt.event.RuleID)
			}
			if unmarshaled.ClientID != tt.event.ClientID {
				t.Errorf("ClientID = %v, want %v", unmarshaled.ClientID, tt.event.ClientID)
			}
			if unmarshaled.Action != tt.event.Action {
				t.Errorf("Action = %v, want %v", unmarshaled.Action, tt.event.Action)
			}
			if unmarshaled.Version != tt.event.Version {
				t.Errorf("Version = %v, want %v", unmarshaled.Version, tt.event.Version)
			}
			if unmarshaled.UpdatedAt != tt.event.UpdatedAt {
				t.Errorf("UpdatedAt = %v, want %v", unmarshaled.UpdatedAt, tt.event.UpdatedAt)
			}
			if unmarshaled.SchemaVersion != tt.event.SchemaVersion {
				t.Errorf("SchemaVersion = %v, want %v", unmarshaled.SchemaVersion, tt.event.SchemaVersion)
			}
		})
	}
}

// TestActionConstants tests that action constants are defined correctly.
func TestActionConstants(t *testing.T) {
	if ActionCreated != "CREATED" {
		t.Errorf("ActionCreated = %v, want CREATED", ActionCreated)
	}
	if ActionUpdated != "UPDATED" {
		t.Errorf("ActionUpdated = %v, want UPDATED", ActionUpdated)
	}
	if ActionDeleted != "DELETED" {
		t.Errorf("ActionDeleted = %v, want DELETED", ActionDeleted)
	}
	if ActionDisabled != "DISABLED" {
		t.Errorf("ActionDisabled = %v, want DISABLED", ActionDisabled)
	}
}
