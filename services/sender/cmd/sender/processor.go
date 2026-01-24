package main

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"

	"sender/internal/consumer"
	"sender/internal/database"
	"sender/internal/events"
	"sender/internal/sender"

	"github.com/afikmenashe/alerting-platform/pkg/metrics"
)

const workerCount = 10

// processNotifications reads notification ready events from Kafka and processes them concurrently.
// Rate limiting for email providers is handled at the email sender level.
func processNotifications(ctx context.Context, kafkaConsumer *consumer.Consumer, db *database.DB, notifSender *sender.Sender, m *metrics.Collector) error {
	slog.Info("Starting notification processing loop", "workers", workerCount)

	type work struct {
		ready *events.NotificationReady
		msg   *kafka.Message
	}

	jobs := make(chan work, workerCount*2)
	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				processOne(ctx, db, notifSender, kafkaConsumer, job.ready, job.msg, m)
			}
		}()
	}

	// Read messages and dispatch to workers
	for {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			slog.Info("Notification processing loop stopped")
			return nil
		default:
			ready, msg, err := kafkaConsumer.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					close(jobs)
					wg.Wait()
					return nil
				}
				slog.Error("Failed to read notification ready event", "error", err)
				continue
			}
			if m != nil {
				m.RecordReceived()
			}
			jobs <- work{ready: ready, msg: msg}
		}
	}
}

// processOne handles a single notification: fetch, send, update status, commit.
func processOne(ctx context.Context, db *database.DB, notifSender *sender.Sender, kafkaConsumer *consumer.Consumer, ready *events.NotificationReady, msg *kafka.Message, m *metrics.Collector) {
	startTime := time.Now()

	notification, err := db.GetNotification(ctx, ready.NotificationID)
	if err != nil {
		slog.Error("Failed to fetch notification", "notification_id", ready.NotificationID, "error", err)
		if m != nil {
			m.RecordError()
		}
		return
	}

	// Skip if already processed (idempotency check)
	if notification.Status == "SENT" || notification.Status == "FAILED" {
		slog.Debug("Notification already processed, skipping",
			"notification_id", ready.NotificationID,
			"status", notification.Status,
		)
		if m != nil {
			m.IncrementCustom("notifications_skipped")
		}
		if err := kafkaConsumer.CommitMessage(ctx, msg); err != nil {
			slog.Error("Failed to commit offset", "error", err)
		}
		return
	}

	endpoints, err := db.GetEndpointsByRuleIDs(ctx, notification.RuleIDs)
	if err != nil {
		slog.Error("Failed to fetch endpoints", "notification_id", ready.NotificationID, "error", err)
		if m != nil {
			m.RecordError()
		}
		return
	}

	if err := notifSender.SendNotification(ctx, notification, endpoints); err != nil {
		slog.Error("Failed to send notification", "notification_id", ready.NotificationID, "error", err)

		// Mark as FAILED (dead letter queue pattern - notification can be retried later)
		if updateErr := db.UpdateNotificationStatus(ctx, ready.NotificationID, "FAILED"); updateErr != nil {
			slog.Error("Failed to mark notification as failed",
				"notification_id", ready.NotificationID,
				"error", updateErr,
			)
			if m != nil {
				m.RecordError()
			}
			// Don't commit - will retry on redelivery
			return
		}

		if m != nil {
			m.RecordProcessed(time.Since(startTime)) // Record latency even for failures
			m.RecordError()
			m.IncrementCustom("notifications_failed")
		}

		slog.Warn("Notification marked as FAILED (DLQ)",
			"notification_id", ready.NotificationID,
			"alert_id", ready.AlertID,
			"client_id", ready.ClientID,
			"error", err,
		)

		// Commit offset - we've handled this notification (by marking it failed)
		if err := kafkaConsumer.CommitMessage(ctx, msg); err != nil {
			slog.Error("Failed to commit offset", "error", err)
		}
		return
	}

	if err := db.UpdateNotificationStatus(ctx, ready.NotificationID, "SENT"); err != nil {
		slog.Error("Failed to update notification status", "notification_id", ready.NotificationID, "error", err)
		if m != nil {
			m.RecordError()
		}
		return
	}

	if m != nil {
		m.RecordProcessed(time.Since(startTime))
		m.RecordPublished() // Count as "published" when successfully sent
		m.IncrementCustom("notifications_sent")
	}

	slog.Info("Successfully sent notification",
		"notification_id", ready.NotificationID,
		"alert_id", ready.AlertID,
		"client_id", ready.ClientID,
		"rule_ids", notification.RuleIDs,
	)

	if err := kafkaConsumer.CommitMessage(ctx, msg); err != nil {
		slog.Error("Failed to commit offset", "notification_id", ready.NotificationID, "error", err)
	}
}
