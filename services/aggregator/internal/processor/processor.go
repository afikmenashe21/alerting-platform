// Package processor provides notification aggregation processing orchestration.
// It handles consuming matched alerts, inserting them idempotently, and publishing notification ready events.
package processor

import (
	"context"
	"log/slog"

	"aggregator/internal/consumer"
	"aggregator/internal/database"
	"aggregator/internal/events"
	"aggregator/internal/producer"
)

// Processor orchestrates notification aggregation and deduplication.
type Processor struct {
	consumer *consumer.Consumer
	producer *producer.Producer
	db       *database.DB
}

// NewProcessor creates a new notification aggregation processor.
func NewProcessor(consumer *consumer.Consumer, producer *producer.Producer, db *database.DB) *Processor {
	return &Processor{
		consumer: consumer,
		producer: producer,
		db:       db,
	}
}

// ProcessNotifications continuously reads matched alerts from Kafka, inserts them
// idempotently into the database, and publishes notification ready events for new notifications.
func (p *Processor) ProcessNotifications(ctx context.Context) error {
	slog.Info("Starting notification processing loop")

	for {
		select {
		case <-ctx.Done():
			slog.Info("Notification processing loop stopped")
			return nil
		default:
			// Read matched alert from Kafka
			matched, msg, err := p.consumer.ReadMessage(ctx)
			if err != nil {
				// Check if context was cancelled
				if ctx.Err() != nil {
					return nil
				}
				slog.Error("Failed to read matched alert", "error", err)
				// Continue processing other messages
				continue
			}

			slog.Debug("Received matched alert",
				"alert_id", matched.AlertID,
				"client_id", matched.ClientID,
				"rule_ids", matched.RuleIDs,
			)

			// Insert notification idempotently
			// This is the dedupe boundary: unique constraint on (client_id, alert_id)
			notificationID, err := p.db.InsertNotificationIdempotent(
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
				// Don't commit offset on error - Kafka will redeliver
				continue
			}

			// Only emit notification ready if a new notification was created
			if notificationID != nil {
				// Build notification ready event
				ready := events.NewNotificationReady(matched, *notificationID)

				// Publish notification ready event
				if err := p.producer.Publish(ctx, ready); err != nil {
					slog.Error("Failed to publish notification ready event",
						"notification_id", *notificationID,
						"alert_id", matched.AlertID,
						"client_id", matched.ClientID,
						"error", err,
					)
					// Don't commit offset on error - Kafka will redeliver
					continue
				}

				slog.Info("Processed new notification",
					"notification_id", *notificationID,
					"alert_id", matched.AlertID,
					"client_id", matched.ClientID,
					"rule_ids", matched.RuleIDs,
				)
			} else {
				slog.Debug("Notification already exists, skipping emit",
					"alert_id", matched.AlertID,
					"client_id", matched.ClientID,
				)
			}

			// Commit offset only after successful DB insert and (if applicable) successful publish
			// This ensures at-least-once semantics: if we crash before commit, Kafka will redeliver
			if err := p.consumer.CommitMessage(ctx, msg); err != nil {
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
