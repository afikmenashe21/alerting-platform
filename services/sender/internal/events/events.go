// Package events defines the event structures for notifications.ready topic.
package events

// NotificationReady represents a notification ready event from the notifications.ready topic.
// This event is emitted by the aggregator when a new notification is created.
type NotificationReady struct {
	NotificationID string `json:"notification_id"`
	ClientID       string `json:"client_id"`
	AlertID        string `json:"alert_id"`
	SchemaVersion  int    `json:"schema_version"`
}
