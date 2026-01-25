// Package events provides tests for event types and conversions.
package events

import (
	"testing"

	protocommon "github.com/afikmenashe/alerting-platform/pkg/proto/common"
)

func TestToProtoAction(t *testing.T) {
	tests := []struct {
		action   string
		expected protocommon.RuleAction
	}{
		{ActionCreated, protocommon.RuleAction_RULE_ACTION_CREATED},
		{ActionUpdated, protocommon.RuleAction_RULE_ACTION_UPDATED},
		{ActionDeleted, protocommon.RuleAction_RULE_ACTION_DELETED},
		{ActionDisabled, protocommon.RuleAction_RULE_ACTION_DISABLED},
		{"UNKNOWN", protocommon.RuleAction_RULE_ACTION_UNSPECIFIED},
		{"", protocommon.RuleAction_RULE_ACTION_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			got := ToProtoAction(tt.action)
			if got != tt.expected {
				t.Errorf("ToProtoAction(%q) = %v, want %v", tt.action, got, tt.expected)
			}
		})
	}
}
