// Package events defines the event structures for alerts.matched and notifications.ready topics.
package events

// AlertMatched represents a matched alert event from the alerts.matched topic.
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

// NotificationReady represents a notification ready event to be published to notifications.ready topic.
// Emitted only for newly created notifications (after successful idempotent insert).
type NotificationReady struct {
	NotificationID string `json:"notification_id"`
	ClientID       string `json:"client_id"`
	AlertID        string `json:"alert_id"`
	SchemaVersion  int    `json:"schema_version"`
}
