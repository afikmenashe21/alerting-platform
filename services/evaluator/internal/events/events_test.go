package events

import (
	"encoding/json"
	"testing"
)

func TestAlertNew_JSON(t *testing.T) {
	tests := []struct {
		name  string
		alert AlertNew
		want  string
	}{
		{
			name: "complete alert",
			alert: AlertNew{
				AlertID:       "alert-123",
				SchemaVersion: 1,
				EventTS:       1234567890,
				Severity:      "HIGH",
				Source:        "service-a",
				Name:          "disk-full",
				Context: map[string]string{
					"disk": "/dev/sda1",
					"usage": "95%",
				},
			},
			want: `{"alert_id":"alert-123","schema_version":1,"event_ts":1234567890,"severity":"HIGH","source":"service-a","name":"disk-full","context":{"disk":"/dev/sda1","usage":"95%"}}`,
		},
		{
			name: "alert without context",
			alert: AlertNew{
				AlertID:       "alert-456",
				SchemaVersion: 1,
				EventTS:       1234567890,
				Severity:      "LOW",
				Source:        "service-b",
				Name:          "cpu-high",
			},
			want: `{"alert_id":"alert-456","schema_version":1,"event_ts":1234567890,"severity":"LOW","source":"service-b","name":"cpu-high"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			got, err := json.Marshal(tt.alert)
			if err != nil {
				t.Fatalf("AlertNew.MarshalJSON() error = %v", err)
			}

			// Normalize JSON (remove whitespace)
			var gotObj, wantObj map[string]interface{}
			if err := json.Unmarshal(got, &gotObj); err != nil {
				t.Fatalf("Failed to unmarshal got: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.want), &wantObj); err != nil {
				t.Fatalf("Failed to unmarshal want: %v", err)
			}

			// Compare normalized JSON
			gotNormalized, _ := json.Marshal(gotObj)
			wantNormalized, _ := json.Marshal(wantObj)
			if string(gotNormalized) != string(wantNormalized) {
				t.Errorf("AlertNew.MarshalJSON() = %v, want %v", string(gotNormalized), string(wantNormalized))
			}

			// Test unmarshaling
			var unmarshaled AlertNew
			if err := json.Unmarshal([]byte(tt.want), &unmarshaled); err != nil {
				t.Fatalf("AlertNew.UnmarshalJSON() error = %v", err)
			}

			if unmarshaled.AlertID != tt.alert.AlertID {
				t.Errorf("AlertNew.UnmarshalJSON() AlertID = %v, want %v", unmarshaled.AlertID, tt.alert.AlertID)
			}
			if unmarshaled.Severity != tt.alert.Severity {
				t.Errorf("AlertNew.UnmarshalJSON() Severity = %v, want %v", unmarshaled.Severity, tt.alert.Severity)
			}
		})
	}
}

func TestAlertMatched_JSON(t *testing.T) {
	tests := []struct {
		name  string
		alert AlertMatched
		want  string
	}{
		{
			name: "complete matched alert",
			alert: AlertMatched{
				AlertID:       "alert-123",
				SchemaVersion: 1,
				EventTS:       1234567890,
				Severity:      "HIGH",
				Source:        "service-a",
				Name:          "disk-full",
				Context: map[string]string{
					"disk": "/dev/sda1",
				},
				ClientID: "client-1",
				RuleIDs: []string{"rule-1", "rule-2"},
			},
			want: `{"alert_id":"alert-123","schema_version":1,"event_ts":1234567890,"severity":"HIGH","source":"service-a","name":"disk-full","context":{"disk":"/dev/sda1"},"client_id":"client-1","rule_ids":["rule-1","rule-2"]}`,
		},
		{
			name: "matched alert without context",
			alert: AlertMatched{
				AlertID:       "alert-456",
				SchemaVersion: 1,
				EventTS:       1234567890,
				Severity:      "LOW",
				Source:        "service-b",
				Name:          "cpu-high",
				ClientID:      "client-2",
				RuleIDs:       []string{"rule-3"},
			},
			want: `{"alert_id":"alert-456","schema_version":1,"event_ts":1234567890,"severity":"LOW","source":"service-b","name":"cpu-high","client_id":"client-2","rule_ids":["rule-3"]}`,
		},
		{
			name: "matched alert with empty rule_ids",
			alert: AlertMatched{
				AlertID:       "alert-789",
				SchemaVersion: 1,
				EventTS:       1234567890,
				Severity:      "MEDIUM",
				Source:        "service-c",
				Name:          "memory-high",
				ClientID:      "client-3",
				RuleIDs:       []string{},
			},
			want: `{"alert_id":"alert-789","schema_version":1,"event_ts":1234567890,"severity":"MEDIUM","source":"service-c","name":"memory-high","client_id":"client-3","rule_ids":[]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			got, err := json.Marshal(tt.alert)
			if err != nil {
				t.Fatalf("AlertMatched.MarshalJSON() error = %v", err)
			}

			// Normalize JSON
			var gotObj, wantObj map[string]interface{}
			if err := json.Unmarshal(got, &gotObj); err != nil {
				t.Fatalf("Failed to unmarshal got: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.want), &wantObj); err != nil {
				t.Fatalf("Failed to unmarshal want: %v", err)
			}

			gotNormalized, _ := json.Marshal(gotObj)
			wantNormalized, _ := json.Marshal(wantObj)
			if string(gotNormalized) != string(wantNormalized) {
				t.Errorf("AlertMatched.MarshalJSON() = %v, want %v", string(gotNormalized), string(wantNormalized))
			}

			// Test unmarshaling
			var unmarshaled AlertMatched
			if err := json.Unmarshal([]byte(tt.want), &unmarshaled); err != nil {
				t.Fatalf("AlertMatched.UnmarshalJSON() error = %v", err)
			}

			if unmarshaled.AlertID != tt.alert.AlertID {
				t.Errorf("AlertMatched.UnmarshalJSON() AlertID = %v, want %v", unmarshaled.AlertID, tt.alert.AlertID)
			}
			if unmarshaled.ClientID != tt.alert.ClientID {
				t.Errorf("AlertMatched.UnmarshalJSON() ClientID = %v, want %v", unmarshaled.ClientID, tt.alert.ClientID)
			}
			if len(unmarshaled.RuleIDs) != len(tt.alert.RuleIDs) {
				t.Errorf("AlertMatched.UnmarshalJSON() RuleIDs length = %v, want %v", len(unmarshaled.RuleIDs), len(tt.alert.RuleIDs))
			}
		})
	}
}

func TestRuleChanged_JSON(t *testing.T) {
	tests := []struct {
		name  string
		rule  RuleChanged
		want  string
	}{
		{
			name: "rule created",
			rule: RuleChanged{
				RuleID:        "rule-1",
				ClientID:      "client-1",
				Action:        "CREATED",
				Version:       1,
				UpdatedAt:     1234567890,
				SchemaVersion: 1,
			},
			want: `{"rule_id":"rule-1","client_id":"client-1","action":"CREATED","version":1,"updated_at":1234567890,"schema_version":1}`,
		},
		{
			name: "rule updated",
			rule: RuleChanged{
				RuleID:        "rule-2",
				ClientID:      "client-2",
				Action:        "UPDATED",
				Version:       2,
				UpdatedAt:     1234567891,
				SchemaVersion: 1,
			},
			want: `{"rule_id":"rule-2","client_id":"client-2","action":"UPDATED","version":2,"updated_at":1234567891,"schema_version":1}`,
		},
		{
			name: "rule deleted",
			rule: RuleChanged{
				RuleID:        "rule-3",
				ClientID:      "client-3",
				Action:        "DELETED",
				Version:       3,
				UpdatedAt:     1234567892,
				SchemaVersion: 1,
			},
			want: `{"rule_id":"rule-3","client_id":"client-3","action":"DELETED","version":3,"updated_at":1234567892,"schema_version":1}`,
		},
		{
			name: "rule disabled",
			rule: RuleChanged{
				RuleID:        "rule-4",
				ClientID:      "client-4",
				Action:        "DISABLED",
				Version:       4,
				UpdatedAt:     1234567893,
				SchemaVersion: 1,
			},
			want: `{"rule_id":"rule-4","client_id":"client-4","action":"DISABLED","version":4,"updated_at":1234567893,"schema_version":1}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			got, err := json.Marshal(tt.rule)
			if err != nil {
				t.Fatalf("RuleChanged.MarshalJSON() error = %v", err)
			}

			// Normalize JSON
			var gotObj, wantObj map[string]interface{}
			if err := json.Unmarshal(got, &gotObj); err != nil {
				t.Fatalf("Failed to unmarshal got: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.want), &wantObj); err != nil {
				t.Fatalf("Failed to unmarshal want: %v", err)
			}

			gotNormalized, _ := json.Marshal(gotObj)
			wantNormalized, _ := json.Marshal(wantObj)
			if string(gotNormalized) != string(wantNormalized) {
				t.Errorf("RuleChanged.MarshalJSON() = %v, want %v", string(gotNormalized), string(wantNormalized))
			}

			// Test unmarshaling
			var unmarshaled RuleChanged
			if err := json.Unmarshal([]byte(tt.want), &unmarshaled); err != nil {
				t.Fatalf("RuleChanged.UnmarshalJSON() error = %v", err)
			}

			if unmarshaled.RuleID != tt.rule.RuleID {
				t.Errorf("RuleChanged.UnmarshalJSON() RuleID = %v, want %v", unmarshaled.RuleID, tt.rule.RuleID)
			}
			if unmarshaled.Action != tt.rule.Action {
				t.Errorf("RuleChanged.UnmarshalJSON() Action = %v, want %v", unmarshaled.Action, tt.rule.Action)
			}
		})
	}
}
