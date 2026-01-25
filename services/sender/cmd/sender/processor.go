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
	"sender/internal/metrics"
	"sender/internal/sender"
)

const workerCount = 10

// work represents a unit of work for the worker pool.
type work struct {
	ready *events.NotificationReady
	msg   *kafka.Message
}

// processorDeps holds all dependencies needed for notification processing.
// This makes testing and dependency injection cleaner.
type processorDeps struct {
	consumer *consumer.Consumer
	db       *database.DB
	sender   *sender.Sender
	metrics  metrics.Recorder
}

// processNotifications reads notification ready events from Kafka and processes them concurrently.
// Rate limiting for email providers is handled at the email sender level.
func processNotifications(ctx context.Context, kafkaConsumer *consumer.Consumer, db *database.DB, notifSender *sender.Sender, m metrics.Recorder) error {
	slog.Info("Starting notification processing loop", "workers", workerCount)

	deps := &processorDeps{
		consumer: kafkaConsumer,
		db:       db,
		sender:   notifSender,
		metrics:  m,
	}

	jobs := make(chan work, workerCount*2)
	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go runWorker(ctx, deps, jobs, &wg)
	}

	// Read messages and dispatch to workers
	dispatchMessages(ctx, deps, jobs)

	close(jobs)
	wg.Wait()
	slog.Info("Notification processing loop stopped")
	return nil
}

// runWorker processes jobs from the channel until it's closed.
func runWorker(ctx context.Context, deps *processorDeps, jobs <-chan work, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		processOne(ctx, deps, job.ready, job.msg)
	}
}

// dispatchMessages reads messages from Kafka and dispatches them to workers.
func dispatchMessages(ctx context.Context, deps *processorDeps, jobs chan<- work) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			ready, msg, err := deps.consumer.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				slog.Error("Failed to read notification ready event", "error", err)
				continue
			}
			deps.metrics.RecordReceived()
			jobs <- work{ready: ready, msg: msg}
		}
	}
}

// processOne handles a single notification: fetch, send, update status, commit.
func processOne(ctx context.Context, deps *processorDeps, ready *events.NotificationReady, msg *kafka.Message) {
	startTime := time.Now()

	// Fetch notification from database
	notification, err := deps.db.GetNotification(ctx, ready.NotificationID)
	if err != nil {
		logAndRecordError(deps.metrics, "Failed to fetch notification",
			"notification_id", ready.NotificationID, "error", err)
		return
	}

	// Skip if already processed (idempotency check)
	if isAlreadyProcessed(notification.Status) {
		handleAlreadyProcessed(ctx, deps, ready, msg)
		return
	}

	// Fetch endpoints for the notification's rules
	endpoints, err := deps.db.GetEndpointsByRuleIDs(ctx, notification.RuleIDs)
	if err != nil {
		logAndRecordError(deps.metrics, "Failed to fetch endpoints",
			"notification_id", ready.NotificationID, "error", err)
		return
	}

	// Attempt to send the notification
	if err := deps.sender.SendNotification(ctx, notification, endpoints); err != nil {
		handleSendFailure(ctx, deps, ready, notification, msg, startTime, err)
		return
	}

	// Update status to SENT and commit
	handleSendSuccess(ctx, deps, ready, notification, msg, startTime)
}

// isAlreadyProcessed checks if a notification has already been processed.
func isAlreadyProcessed(status string) bool {
	return database.NotificationStatus(status).IsTerminal()
}

// handleAlreadyProcessed handles the case where notification was already processed.
func handleAlreadyProcessed(ctx context.Context, deps *processorDeps, ready *events.NotificationReady, msg *kafka.Message) {
	slog.Debug("Notification already processed, skipping",
		"notification_id", ready.NotificationID,
		"status", "already_processed",
	)
	deps.metrics.RecordSkipped()
	commitOffset(ctx, deps.consumer, msg)
}

// handleSendFailure handles the case where sending a notification failed.
func handleSendFailure(ctx context.Context, deps *processorDeps, ready *events.NotificationReady, notification *database.Notification, msg *kafka.Message, startTime time.Time, sendErr error) {
	slog.Error("Failed to send notification",
		"notification_id", ready.NotificationID,
		"error", sendErr,
	)

	// Mark as FAILED (dead letter queue pattern - notification can be retried later)
	if err := deps.db.UpdateNotificationStatus(ctx, ready.NotificationID, database.StatusFailed.String()); err != nil {
		logAndRecordError(deps.metrics, "Failed to mark notification as failed",
			"notification_id", ready.NotificationID, "error", err)
		// Don't commit - will retry on redelivery
		return
	}

	deps.metrics.RecordProcessed(time.Since(startTime))
	deps.metrics.RecordError()
	deps.metrics.RecordFailed()

	slog.Warn("Notification marked as FAILED (DLQ)",
		"notification_id", ready.NotificationID,
		"alert_id", ready.AlertID,
		"client_id", ready.ClientID,
		"error", sendErr,
	)

	// Commit offset - we've handled this notification (by marking it failed)
	commitOffset(ctx, deps.consumer, msg)
}

// handleSendSuccess handles the case where sending a notification succeeded.
func handleSendSuccess(ctx context.Context, deps *processorDeps, ready *events.NotificationReady, notification *database.Notification, msg *kafka.Message, startTime time.Time) {
	if err := deps.db.UpdateNotificationStatus(ctx, ready.NotificationID, database.StatusSent.String()); err != nil {
		logAndRecordError(deps.metrics, "Failed to update notification status",
			"notification_id", ready.NotificationID, "error", err)
		return
	}

	deps.metrics.RecordProcessed(time.Since(startTime))
	deps.metrics.RecordPublished()
	deps.metrics.RecordSent()

	slog.Info("Successfully sent notification",
		"notification_id", ready.NotificationID,
		"alert_id", ready.AlertID,
		"client_id", ready.ClientID,
		"rule_ids", notification.RuleIDs,
	)

	commitOffset(ctx, deps.consumer, msg)
}

// commitOffset commits the Kafka offset for the given message.
func commitOffset(ctx context.Context, c *consumer.Consumer, msg *kafka.Message) {
	if err := c.CommitMessage(ctx, msg); err != nil {
		slog.Error("Failed to commit offset", "error", err)
	}
}

// logAndRecordError logs an error and records it in metrics.
func logAndRecordError(m metrics.Recorder, msg string, args ...any) {
	slog.Error(msg, args...)
	m.RecordError()
}
