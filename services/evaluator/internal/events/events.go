// Package events defines the event structures for alerts.new and alerts.matched topics.
package events

// AlertNew represents an alert event from the alerts.new topic.
type AlertNew struct {
	AlertID       string            `json:"alert_id"`
	SchemaVersion int               `json:"schema_version"`
	EventTS       int64             `json:"event_ts"`
	Severity      string            `json:"severity"`
	Source        string            `json:"source"`
	Name          string            `json:"name"`
	Context       map[string]string `json:"context,omitempty"`
}

// AlertMatched represents a matched alert event to be published to alerts.matched topic.
// One message per client_id, containing the alert and the rule_ids that matched for that client.
type AlertMatched struct {
	AlertID       string            `json:"alert_id"`
	SchemaVersion int               `json:"schema_version"`
	EventTS       int64             `json:"event_ts"`
	Severity      string            `json:"severity"`
	Source        string            `json:"source"`
	Name          string            `json:"name"`
	Context       map[string]string `json:"context,omitempty"`
	ClientID      string            `json:"client_id"` // The client this message is for
	RuleIDs       []string          `json:"rule_ids"` // All rule IDs that matched for this client
}

// NewAlertMatched creates a new AlertMatched event from an AlertNew event for a specific client.
func NewAlertMatched(alert *AlertNew, clientID string, ruleIDs []string) *AlertMatched {
	return &AlertMatched{
		AlertID:       alert.AlertID,
		SchemaVersion: alert.SchemaVersion,
		EventTS:       alert.EventTS,
		Severity:      alert.Severity,
		Source:        alert.Source,
		Name:          alert.Name,
		Context:       alert.Context,
		ClientID:      clientID,
		RuleIDs:       ruleIDs,
	}
}

// RuleChanged represents a rule change event from the rule.changed topic.
type RuleChanged struct {
	RuleID        string `json:"rule_id"`
	ClientID      string `json:"client_id"`
	Action        string `json:"action"` // CREATED, UPDATED, DELETED, DISABLED
	Version       int    `json:"version"`
	UpdatedAt     int64  `json:"updated_at"` // Unix timestamp
	SchemaVersion int    `json:"schema_version"`
}
