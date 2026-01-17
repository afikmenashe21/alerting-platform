// Package strategy defines the interface for notification sending strategies.
package strategy

import (
	"context"

	"sender/internal/database"
)

// NotificationSender is the interface that all notification sending strategies must implement.
type NotificationSender interface {
	// Send sends a notification to the specified endpoint value.
	// The endpoint value format depends on the sender type:
	//   - Email: email address(es) as comma-separated string
	//   - Slack: webhook URL
	//   - Webhook: webhook URL
	Send(ctx context.Context, endpointValue string, notification *database.Notification) error

	// Type returns the endpoint type this sender handles (e.g., "email", "slack", "webhook").
	Type() string
}

// Registry manages notification sender strategies.
type Registry struct {
	senders map[string]NotificationSender
}

// NewRegistry creates a new sender registry.
func NewRegistry() *Registry {
	return &Registry{
		senders: make(map[string]NotificationSender),
	}
}

// Register registers a sender strategy.
func (r *Registry) Register(sender NotificationSender) {
	r.senders[sender.Type()] = sender
}

// Get retrieves a sender strategy by type.
func (r *Registry) Get(senderType string) (NotificationSender, bool) {
	sender, ok := r.senders[senderType]
	return sender, ok
}

// List returns all registered sender types.
func (r *Registry) List() []string {
	types := make([]string, 0, len(r.senders))
	for t := range r.senders {
		types = append(types, t)
	}
	return types
}
