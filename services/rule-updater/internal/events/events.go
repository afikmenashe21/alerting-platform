// Package events defines the event structures for rule-updater.
package events

import "fmt"

// Action represents the type of change that occurred to a rule.
type Action string

// Valid actions for RuleChanged events.
const (
	ActionCreated  Action = "CREATED"
	ActionUpdated  Action = "UPDATED"
	ActionDeleted  Action = "DELETED"
	ActionDisabled Action = "DISABLED"
)

// IsAdditive returns true if the action adds or updates a rule (requires DB lookup).
func (a Action) IsAdditive() bool {
	return a == ActionCreated || a == ActionUpdated
}

// IsRemoval returns true if the action removes a rule from the snapshot.
func (a Action) IsRemoval() bool {
	return a == ActionDeleted || a == ActionDisabled
}

// IsValid returns true if the action is a known valid action.
func (a Action) IsValid() bool {
	switch a {
	case ActionCreated, ActionUpdated, ActionDeleted, ActionDisabled:
		return true
	default:
		return false
	}
}

// String returns the string representation of the action.
func (a Action) String() string {
	return string(a)
}

// RuleChanged represents a rule change event published to rule.changed topic.
// This matches the structure from rule-service.
type RuleChanged struct {
	RuleID        string `json:"rule_id"`
	ClientID      string `json:"client_id"`
	Action        Action `json:"action"`
	Version       int    `json:"version"`
	UpdatedAt     int64  `json:"updated_at"` // Unix timestamp
	SchemaVersion int    `json:"schema_version"`
}

// Validate checks that the event has valid required fields.
func (e *RuleChanged) Validate() error {
	if e.RuleID == "" {
		return fmt.Errorf("rule_id is required")
	}
	if !e.Action.IsValid() {
		return fmt.Errorf("invalid action: %s", e.Action)
	}
	return nil
}
