// Package processor provides notification aggregation processing orchestration.
// It handles consuming matched alerts, inserting them idempotently, and publishing notification ready events.
package processor

import (
	"context"
	"log/slog"
	"time"

	"aggregator/internal/consumer"
	"aggregator/internal/database"
	"aggregator/internal/events"
	"aggregator/internal/producer"

	"github.com/afikmenashe/alerting-platform/pkg/metrics"
)

// Processor orchestrates notification aggregation and deduplication.
type Processor struct {
	consumer *consumer.Consumer
	producer *producer.Producer
	db       *database.DB
	metrics  *metrics.Collector
}

// NewProcessor creates a new notification aggregation processor (without metrics).
func NewProcessor(consumer *consumer.Consumer, producer *producer.Producer, db *database.DB) *Processor {
	return &Processor{
		consumer: consumer,
		producer: producer,
		db:       db,
		metrics:  nil,
	}
}

// NewProcessorWithMetrics creates a processor with shared metrics collector.
func NewProcessorWithMetrics(consumer *consumer.Consumer, producer *producer.Producer, db *database.DB, m *metrics.Collector) *Processor {
	return &Processor{
		consumer: consumer,
		producer: producer,
		db:       db,
		metrics:  m,
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

			if p.metrics != nil {
				p.metrics.RecordReceived()
			}

			startTime := time.Now()

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
				if p.metrics != nil {
					p.metrics.RecordError()
				}
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
					if p.metrics != nil {
						p.metrics.RecordError()
					}
					// Don't commit offset on error - Kafka will redeliver
					continue
				}

				if p.metrics != nil {
					p.metrics.RecordPublished()
					p.metrics.IncrementCustom("notifications_created")
				}

				slog.Info("Processed new notification",
					"notification_id", *notificationID,
					"alert_id", matched.AlertID,
					"client_id", matched.ClientID,
					"rule_ids", matched.RuleIDs,
				)
			} else {
				if p.metrics != nil {
					p.metrics.IncrementCustom("notifications_deduplicated")
				}
				slog.Debug("Notification already exists, skipping emit",
					"alert_id", matched.AlertID,
					"client_id", matched.ClientID,
				)
			}

			if p.metrics != nil {
				p.metrics.RecordProcessed(time.Since(startTime))
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
