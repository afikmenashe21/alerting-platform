package events

import "testing"

func TestAction_IsAdditive(t *testing.T) {
	tests := []struct {
		action   Action
		expected bool
	}{
		{ActionCreated, true},
		{ActionUpdated, true},
		{ActionDeleted, false},
		{ActionDisabled, false},
		{Action("UNKNOWN"), false},
		{Action(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			if got := tt.action.IsAdditive(); got != tt.expected {
				t.Errorf("Action(%q).IsAdditive() = %v, want %v", tt.action, got, tt.expected)
			}
		})
	}
}

func TestAction_IsRemoval(t *testing.T) {
	tests := []struct {
		action   Action
		expected bool
	}{
		{ActionCreated, false},
		{ActionUpdated, false},
		{ActionDeleted, true},
		{ActionDisabled, true},
		{Action("UNKNOWN"), false},
		{Action(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			if got := tt.action.IsRemoval(); got != tt.expected {
				t.Errorf("Action(%q).IsRemoval() = %v, want %v", tt.action, got, tt.expected)
			}
		})
	}
}

func TestAction_IsValid(t *testing.T) {
	tests := []struct {
		action   Action
		expected bool
	}{
		{ActionCreated, true},
		{ActionUpdated, true},
		{ActionDeleted, true},
		{ActionDisabled, true},
		{Action("UNKNOWN"), false},
		{Action(""), false},
		{Action("created"), false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			if got := tt.action.IsValid(); got != tt.expected {
				t.Errorf("Action(%q).IsValid() = %v, want %v", tt.action, got, tt.expected)
			}
		})
	}
}

func TestAction_String(t *testing.T) {
	tests := []struct {
		action   Action
		expected string
	}{
		{ActionCreated, "CREATED"},
		{ActionUpdated, "UPDATED"},
		{ActionDeleted, "DELETED"},
		{ActionDisabled, "DISABLED"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.action.String(); got != tt.expected {
				t.Errorf("Action.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRuleChanged_Validate(t *testing.T) {
	tests := []struct {
		name    string
		event   RuleChanged
		wantErr bool
	}{
		{
			name: "valid created event",
			event: RuleChanged{
				RuleID:   "rule-1",
				ClientID: "client-1",
				Action:   ActionCreated,
				Version:  1,
			},
			wantErr: false,
		},
		{
			name: "valid deleted event",
			event: RuleChanged{
				RuleID:   "rule-1",
				ClientID: "client-1",
				Action:   ActionDeleted,
				Version:  1,
			},
			wantErr: false,
		},
		{
			name: "empty rule_id",
			event: RuleChanged{
				RuleID:   "",
				ClientID: "client-1",
				Action:   ActionCreated,
				Version:  1,
			},
			wantErr: true,
		},
		{
			name: "invalid action",
			event: RuleChanged{
				RuleID:   "rule-1",
				ClientID: "client-1",
				Action:   Action("INVALID"),
				Version:  1,
			},
			wantErr: true,
		},
		{
			name: "empty action",
			event: RuleChanged{
				RuleID:   "rule-1",
				ClientID: "client-1",
				Action:   Action(""),
				Version:  1,
			},
			wantErr: true,
		},
		{
			name: "empty client_id is allowed",
			event: RuleChanged{
				RuleID:   "rule-1",
				ClientID: "",
				Action:   ActionUpdated,
				Version:  1,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("RuleChanged.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
