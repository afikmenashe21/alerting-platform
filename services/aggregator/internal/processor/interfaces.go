// Package processor provides notification aggregation processing orchestration.
package processor

import (
	"context"

	"aggregator/internal/events"

	"github.com/segmentio/kafka-go"
)

// MessageReader reads matched alert messages from a message queue.
type MessageReader interface {
	// ReadMessage reads the next message and returns the parsed AlertMatched event.
	// Returns the raw message for offset tracking.
	ReadMessage(ctx context.Context) (*events.AlertMatched, *kafka.Message, error)

	// CommitMessage commits the offset for the given message.
	CommitMessage(ctx context.Context, msg *kafka.Message) error

	// Close closes the reader and releases resources.
	Close() error
}

// MessagePublisher publishes notification ready events to a message queue.
type MessagePublisher interface {
	// Publish publishes a notification ready event.
	Publish(ctx context.Context, ready *events.NotificationReady) error

	// Close closes the publisher and releases resources.
	Close() error
}

// NotificationStorage stores notification records for deduplication.
type NotificationStorage interface {
	// InsertNotificationIdempotent inserts a notification with idempotency protection.
	// Returns the notification ID if a new row was inserted, or nil if it already existed.
	InsertNotificationIdempotent(
		ctx context.Context,
		clientID, alertID, severity, source, name string,
		context map[string]string,
		ruleIDs []string,
	) (*string, error)

	// Close closes the storage connection.
	Close() error
}
