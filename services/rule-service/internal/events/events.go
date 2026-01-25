// Package events defines the event structures for rule-service.
package events

import (
	protocommon "github.com/afikmenashe/alerting-platform/pkg/proto/common"
)

// RuleChanged represents a rule change event published to rule.changed topic.
type RuleChanged struct {
	RuleID        string `json:"rule_id"`
	ClientID      string `json:"client_id"`
	Action        string `json:"action"` // CREATED, UPDATED, DELETED, DISABLED
	Version       int    `json:"version"`
	UpdatedAt     int64  `json:"updated_at"` // Unix timestamp
	SchemaVersion int    `json:"schema_version"`
}

// Valid actions for RuleChanged
const (
	ActionCreated  = "CREATED"
	ActionUpdated  = "UPDATED"
	ActionDeleted  = "DELETED"
	ActionDisabled = "DISABLED"
)

// ToProtoAction converts a string action to the protobuf RuleAction enum.
// This centralizes the mapping logic for consistent encoding.
func ToProtoAction(action string) protocommon.RuleAction {
	switch action {
	case ActionCreated:
		return protocommon.RuleAction_RULE_ACTION_CREATED
	case ActionUpdated:
		return protocommon.RuleAction_RULE_ACTION_UPDATED
	case ActionDeleted:
		return protocommon.RuleAction_RULE_ACTION_DELETED
	case ActionDisabled:
		return protocommon.RuleAction_RULE_ACTION_DISABLED
	default:
		return protocommon.RuleAction_RULE_ACTION_UNSPECIFIED
	}
}
