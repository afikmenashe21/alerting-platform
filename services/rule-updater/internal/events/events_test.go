package events

import (
	"encoding/json"
	"testing"
)

func TestRuleChanged_JSONMarshal(t *testing.T) {
	ruleChanged := RuleChanged{
		RuleID:        "rule-123",
		ClientID:      "client-456",
		Action:        ActionCreated,
		Version:       1,
		UpdatedAt:     1234567890,
		SchemaVersion: 1,
	}

	data, err := json.Marshal(ruleChanged)
	if err != nil {
		t.Fatalf("Failed to marshal RuleChanged: %v", err)
	}

	var unmarshaled RuleChanged
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal RuleChanged: %v", err)
	}

	if unmarshaled.RuleID != ruleChanged.RuleID {
		t.Errorf("RuleID = %v, want %v", unmarshaled.RuleID, ruleChanged.RuleID)
	}
	if unmarshaled.ClientID != ruleChanged.ClientID {
		t.Errorf("ClientID = %v, want %v", unmarshaled.ClientID, ruleChanged.ClientID)
	}
	if unmarshaled.Action != ruleChanged.Action {
		t.Errorf("Action = %v, want %v", unmarshaled.Action, ruleChanged.Action)
	}
	if unmarshaled.Version != ruleChanged.Version {
		t.Errorf("Version = %v, want %v", unmarshaled.Version, ruleChanged.Version)
	}
	if unmarshaled.UpdatedAt != ruleChanged.UpdatedAt {
		t.Errorf("UpdatedAt = %v, want %v", unmarshaled.UpdatedAt, ruleChanged.UpdatedAt)
	}
	if unmarshaled.SchemaVersion != ruleChanged.SchemaVersion {
		t.Errorf("SchemaVersion = %v, want %v", unmarshaled.SchemaVersion, ruleChanged.SchemaVersion)
	}
}

func TestRuleChanged_AllActions(t *testing.T) {
	actions := []string{ActionCreated, ActionUpdated, ActionDeleted, ActionDisabled}

	for _, action := range actions {
		ruleChanged := RuleChanged{
			RuleID:        "rule-123",
			ClientID:      "client-456",
			Action:        action,
			Version:       1,
			UpdatedAt:     1234567890,
			SchemaVersion: 1,
		}

		data, err := json.Marshal(ruleChanged)
		if err != nil {
			t.Fatalf("Failed to marshal RuleChanged with action %s: %v", action, err)
		}

		var unmarshaled RuleChanged
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal RuleChanged with action %s: %v", action, err)
		}

		if unmarshaled.Action != action {
			t.Errorf("Action = %v, want %v", unmarshaled.Action, action)
		}
	}
}

func TestRuleChanged_JSONRoundTrip(t *testing.T) {
	tests := []struct {
		name        string
		ruleChanged RuleChanged
	}{
		{
			name: "created action",
			ruleChanged: RuleChanged{
				RuleID:        "rule-1",
				ClientID:      "client-1",
				Action:        ActionCreated,
				Version:       1,
				UpdatedAt:     1000000000,
				SchemaVersion: 1,
			},
		},
		{
			name: "updated action",
			ruleChanged: RuleChanged{
				RuleID:        "rule-2",
				ClientID:      "client-2",
				Action:        ActionUpdated,
				Version:       2,
				UpdatedAt:     2000000000,
				SchemaVersion: 1,
			},
		},
		{
			name: "deleted action",
			ruleChanged: RuleChanged{
				RuleID:        "rule-3",
				ClientID:      "client-3",
				Action:        ActionDeleted,
				Version:       3,
				UpdatedAt:     3000000000,
				SchemaVersion: 1,
			},
		},
		{
			name: "disabled action",
			ruleChanged: RuleChanged{
				RuleID:        "rule-4",
				ClientID:      "client-4",
				Action:        ActionDisabled,
				Version:       4,
				UpdatedAt:     4000000000,
				SchemaVersion: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.ruleChanged)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var unmarshaled RuleChanged
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if unmarshaled.RuleID != tt.ruleChanged.RuleID {
				t.Errorf("RuleID = %v, want %v", unmarshaled.RuleID, tt.ruleChanged.RuleID)
			}
			if unmarshaled.ClientID != tt.ruleChanged.ClientID {
				t.Errorf("ClientID = %v, want %v", unmarshaled.ClientID, tt.ruleChanged.ClientID)
			}
			if unmarshaled.Action != tt.ruleChanged.Action {
				t.Errorf("Action = %v, want %v", unmarshaled.Action, tt.ruleChanged.Action)
			}
			if unmarshaled.Version != tt.ruleChanged.Version {
				t.Errorf("Version = %v, want %v", unmarshaled.Version, tt.ruleChanged.Version)
			}
			if unmarshaled.UpdatedAt != tt.ruleChanged.UpdatedAt {
				t.Errorf("UpdatedAt = %v, want %v", unmarshaled.UpdatedAt, tt.ruleChanged.UpdatedAt)
			}
			if unmarshaled.SchemaVersion != tt.ruleChanged.SchemaVersion {
				t.Errorf("SchemaVersion = %v, want %v", unmarshaled.SchemaVersion, tt.ruleChanged.SchemaVersion)
			}
		})
	}
}
