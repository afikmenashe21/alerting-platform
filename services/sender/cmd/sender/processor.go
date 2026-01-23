package main

import (
	"context"
	"log/slog"
	"sync"

	"github.com/segmentio/kafka-go"

	"sender/internal/consumer"
	"sender/internal/database"
	"sender/internal/events"
	"sender/internal/sender"
)

const workerCount = 10

// processNotifications reads notification ready events from Kafka and processes them concurrently.
func processNotifications(ctx context.Context, kafkaConsumer *consumer.Consumer, db *database.DB, notifSender *sender.Sender) error {
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
				processOne(ctx, db, notifSender, kafkaConsumer, job.ready, job.msg)
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
			jobs <- work{ready: ready, msg: msg}
		}
	}
}

// processOne handles a single notification: fetch, send, update status, commit.
func processOne(ctx context.Context, db *database.DB, notifSender *sender.Sender, kafkaConsumer *consumer.Consumer, ready *events.NotificationReady, msg *kafka.Message) {
	notification, err := db.GetNotification(ctx, ready.NotificationID)
	if err != nil {
		slog.Error("Failed to fetch notification", "notification_id", ready.NotificationID, "error", err)
		return
	}

	if notification.Status == "SENT" {
		if err := kafkaConsumer.CommitMessage(ctx, msg); err != nil {
			slog.Error("Failed to commit offset", "error", err)
		}
		return
	}

	endpoints, err := db.GetEndpointsByRuleIDs(ctx, notification.RuleIDs)
	if err != nil {
		slog.Error("Failed to fetch endpoints", "notification_id", ready.NotificationID, "error", err)
		return
	}

	if err := notifSender.SendNotification(ctx, notification, endpoints); err != nil {
		slog.Error("Failed to send notification", "notification_id", ready.NotificationID, "error", err)
		return
	}

	if err := db.UpdateNotificationStatus(ctx, ready.NotificationID, "SENT"); err != nil {
		slog.Error("Failed to update notification status", "notification_id", ready.NotificationID, "error", err)
		return
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
