// Package processor provides notification aggregation processing orchestration.
// It handles consuming matched alerts, inserting them idempotently, and publishing notification ready events.
package processor

import (
	"context"
	"log/slog"
	"time"

	"aggregator/internal/events"
)

// Processor orchestrates notification aggregation and deduplication.
type Processor struct {
	reader    MessageReader
	publisher MessagePublisher
	storage   NotificationStorage
	metrics   MetricsRecorder
}

// NewProcessor creates a new notification aggregation processor with no-op metrics.
func NewProcessor(reader MessageReader, publisher MessagePublisher, storage NotificationStorage) *Processor {
	return &Processor{
		reader:    reader,
		publisher: publisher,
		storage:   storage,
		metrics:   &NoOpMetrics{},
	}
}

// NewProcessorWithMetrics creates a processor with the provided metrics recorder.
// If m is nil, a no-op implementation is used.
func NewProcessorWithMetrics(reader MessageReader, publisher MessagePublisher, storage NotificationStorage, m MetricsRecorder) *Processor {
	if m == nil {
		m = &NoOpMetrics{}
	}
	return &Processor{
		reader:    reader,
		publisher: publisher,
		storage:   storage,
		metrics:   m,
	}
}

// ProcessNotifications continuously reads matched alerts from the message queue, inserts them
// idempotently into the database, and publishes notification ready events for new notifications.
func (p *Processor) ProcessNotifications(ctx context.Context) error {
	slog.Info("Starting notification processing loop")

	for {
		select {
		case <-ctx.Done():
			slog.Info("Notification processing loop stopped")
			return nil
		default:
			// Read matched alert from message queue
			matched, msg, err := p.reader.ReadMessage(ctx)
			if err != nil {
				// Check if context was cancelled
				if ctx.Err() != nil {
					return nil
				}
				slog.Error("Failed to read matched alert", "error", err)
				continue
			}

			p.metrics.RecordReceived()

			// Process the message; only commit if processing succeeds
			if !p.processMessage(ctx, matched) {
				continue
			}

			// Commit offset only after successful processing
			// This ensures at-least-once semantics: if we crash before commit, message will be redelivered
			if err := p.reader.CommitMessage(ctx, msg); err != nil {
				slog.Error("Failed to commit offset",
					"alert_id", matched.AlertID,
					"client_id", matched.ClientID,
					"error", err,
				)
				// Continue processing - offset will be committed on next interval or retry
			}
		}
	}
}

// processMessage handles a single matched alert: inserts it idempotently
// and publishes a notification ready event if it's new.
// Returns true if processing succeeded and the message should be committed.
func (p *Processor) processMessage(ctx context.Context, matched *events.AlertMatched) bool {
	startTime := time.Now()

	slog.Debug("Received matched alert",
		"alert_id", matched.AlertID,
		"client_id", matched.ClientID,
		"rule_ids", matched.RuleIDs,
	)

	// Insert notification idempotently
	// This is the dedupe boundary: unique constraint on (client_id, alert_id)
	notificationID, err := p.storage.InsertNotificationIdempotent(
		ctx,
		matched.ClientID,
		matched.AlertID,
		matched.Severity,
		matched.Source,
		matched.Name,
		matched.Context,
		matched.RuleIDs,
	)
	if err != nil {
		slog.Error("Failed to insert notification",
			"alert_id", matched.AlertID,
			"client_id", matched.ClientID,
			"error", err,
		)
		p.metrics.RecordError()
		return false
	}

	// Only emit notification ready if a new notification was created
	if notificationID != nil {
		if !p.publishNotification(ctx, matched, *notificationID) {
			return false
		}
	} else {
		p.metrics.IncrementCustom("notifications_deduplicated")
		slog.Debug("Notification already exists, skipping emit",
			"alert_id", matched.AlertID,
			"client_id", matched.ClientID,
		)
	}

	p.metrics.RecordProcessed(time.Since(startTime))
	return true
}

// publishNotification publishes a notification ready event for a newly created notification.
// Returns true if publishing succeeded.
func (p *Processor) publishNotification(ctx context.Context, matched *events.AlertMatched, notificationID string) bool {
	ready := events.NewNotificationReady(matched, notificationID)

	if err := p.publisher.Publish(ctx, ready); err != nil {
		slog.Error("Failed to publish notification ready event",
			"notification_id", notificationID,
			"alert_id", matched.AlertID,
			"client_id", matched.ClientID,
			"error", err,
		)
		p.metrics.RecordError()
		return false
	}

	p.metrics.RecordPublished()
	p.metrics.IncrementCustom("notifications_created")

	slog.Info("Processed new notification",
		"notification_id", notificationID,
		"alert_id", matched.AlertID,
		"client_id", matched.ClientID,
		"rule_ids", matched.RuleIDs,
	)

	return true
}
