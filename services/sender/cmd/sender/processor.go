package main

import (
	"context"
	"log/slog"

	"sender/internal/consumer"
	"sender/internal/database"
	"sender/internal/sender"
)

// processNotifications continuously reads notification ready events from Kafka,
// fetches the notification and endpoints from the database, sends notifications via all channels, and updates status.
func processNotifications(ctx context.Context, kafkaConsumer *consumer.Consumer, db *database.DB, notifSender *sender.Sender) error {
	slog.Info("Starting notification processing loop")

	for {
		select {
		case <-ctx.Done():
			slog.Info("Notification processing loop stopped")
			return nil
		default:
			// Read notification ready event from Kafka
			ready, msg, err := kafkaConsumer.ReadMessage(ctx)
			if err != nil {
				// Check if context was cancelled
				if ctx.Err() != nil {
					return nil
				}
				slog.Error("Failed to read notification ready event", "error", err)
				// Continue processing other messages
				continue
			}

			slog.Debug("Received notification ready event",
				"notification_id", ready.NotificationID,
				"client_id", ready.ClientID,
				"alert_id", ready.AlertID,
			)

			// Fetch notification from database
			notification, err := db.GetNotification(ctx, ready.NotificationID)
			if err != nil {
				slog.Error("Failed to fetch notification",
					"notification_id", ready.NotificationID,
					"error", err,
				)
				// Don't commit offset on error - Kafka will redeliver
				continue
			}

			// Check if already sent (idempotency check)
			if notification.Status == "SENT" {
				slog.Debug("Notification already sent, skipping",
					"notification_id", ready.NotificationID,
				)
				// Commit offset even if already sent (at-least-once semantics)
				if err := kafkaConsumer.CommitMessage(ctx, msg); err != nil {
					slog.Error("Failed to commit offset", "error", err)
				}
				continue
			}

			// Fetch all endpoints (email, slack, webhook) for the rule IDs
			endpoints, err := db.GetEndpointsByRuleIDs(ctx, notification.RuleIDs)
			if err != nil {
				slog.Error("Failed to fetch endpoints",
					"notification_id", ready.NotificationID,
					"rule_ids", notification.RuleIDs,
					"error", err,
				)
				// Don't commit offset on error - Kafka will redeliver
				continue
			}

			// Send notification via all endpoint types (email, slack, webhook)
			if err := notifSender.SendNotification(ctx, notification, endpoints); err != nil {
				slog.Error("Failed to send notification",
					"notification_id", ready.NotificationID,
					"error", err,
				)
				// Don't commit offset on error - Kafka will redeliver
				continue
			}

			// Update notification status to SENT
			if err := db.UpdateNotificationStatus(ctx, ready.NotificationID, "SENT"); err != nil {
				slog.Error("Failed to update notification status",
					"notification_id", ready.NotificationID,
					"error", err,
				)
				// Don't commit offset on error - Kafka will redeliver
				// Note: Email was sent but status not updated. On retry, we'll check status and skip.
				continue
			}

			slog.Info("Successfully sent notification",
				"notification_id", ready.NotificationID,
				"alert_id", ready.AlertID,
				"client_id", ready.ClientID,
				"rule_ids", notification.RuleIDs,
			)

			// Commit offset only after successful send and status update
			// This ensures at-least-once semantics: if we crash before commit, Kafka will redeliver
			if err := kafkaConsumer.CommitMessage(ctx, msg); err != nil {
				slog.Error("Failed to commit offset",
					"notification_id", ready.NotificationID,
					"error", err,
				)
				// Continue processing - offset will be committed on next interval or retry
			}
		}
	}
}
