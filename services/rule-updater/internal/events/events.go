// Package events defines the event structures for rule-updater.
package events

// RuleChanged represents a rule change event published to rule.changed topic.
// This matches the structure from rule-service.
type RuleChanged struct {
	RuleID       string `json:"rule_id"`
	ClientID     string `json:"client_id"`
	Action       string `json:"action"` // CREATED, UPDATED, DELETED, DISABLED
	Version      int    `json:"version"`
	UpdatedAt    int64  `json:"updated_at"` // Unix timestamp
	SchemaVersion int  `json:"schema_version"`
}

// Valid actions for RuleChanged
const (
	ActionCreated  = "CREATED"
	ActionUpdated  = "UPDATED"
	ActionDeleted  = "DELETED"
	ActionDisabled = "DISABLED"
)
