package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"

	"sender/internal/consumer"
	"sender/internal/database"
	"sender/internal/events"
	"sender/internal/sender"

	"github.com/afikmenashe/alerting-platform/pkg/metrics"
)

// getEnvInt reads an environment variable as int with a default value.
func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

// processNotifications reads notification ready events from Kafka and processes them sequentially
// with rate limiting to respect email provider limits (Resend: 2 req/sec, SES: 1 req/sec sandbox).
func processNotifications(ctx context.Context, kafkaConsumer *consumer.Consumer, db *database.DB, notifSender *sender.Sender, m *metrics.Collector) error {
	// Rate limit: emails per second (default 2 for Resend free tier)
	emailsPerSecond := getEnvInt("EMAIL_RATE_LIMIT", 2)
	rateLimitInterval := time.Second / time.Duration(emailsPerSecond)

	slog.Info("Starting notification processing loop",
		"rate_limit", emailsPerSecond,
		"interval_ms", rateLimitInterval.Milliseconds(),
	)

	// Use a ticker for rate limiting
	ticker := time.NewTicker(rateLimitInterval)
	defer ticker.Stop()

	// Process messages sequentially with rate limiting
	for {
		select {
		case <-ctx.Done():
			slog.Info("Notification processing loop stopped")
			return nil
		default:
			ready, msg, err := kafkaConsumer.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				slog.Error("Failed to read notification ready event", "error", err)
				continue
			}
			if m != nil {
				m.RecordReceived()
			}

			// Wait for rate limiter before processing
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				// Rate limit passed, process the notification
			}

			processOne(ctx, db, notifSender, kafkaConsumer, ready, msg, m)
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
