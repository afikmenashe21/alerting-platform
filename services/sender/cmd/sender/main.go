package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"sender/internal/config"
	"sender/internal/consumer"
	"sender/internal/database"
	"sender/internal/sender"
)

func main() {
	// Parse command-line flags
	cfg := &config.Config{}
	flag.StringVar(&cfg.KafkaBrokers, "kafka-brokers", "localhost:9092", "Kafka broker addresses (comma-separated)")
	flag.StringVar(&cfg.NotificationsReadyTopic, "notifications-ready-topic", "notifications.ready", "Kafka topic for ready notifications")
	flag.StringVar(&cfg.ConsumerGroupID, "consumer-group-id", "sender-group", "Kafka consumer group ID")
	flag.StringVar(&cfg.PostgresDSN, "postgres-dsn", "postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable", "PostgreSQL connection string")
	flag.Parse()

	// Set up structured logging
	// Allow DEBUG level via environment variable for troubleshooting
	logLevel := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "DEBUG" || os.Getenv("LOG_LEVEL") == "debug" {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})))

	slog.Info("Starting sender service",
		"kafka_brokers", cfg.KafkaBrokers,
		"notifications_ready_topic", cfg.NotificationsReadyTopic,
		"consumer_group_id", cfg.ConsumerGroupID,
		"postgres_dsn", maskDSN(cfg.PostgresDSN),
	)

	if err := cfg.Validate(); err != nil {
		slog.Error("Invalid configuration", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		slog.Info("Received shutdown signal, shutting down gracefully...")
		cancel()
	}()

	// Initialize database connection
	slog.Info("Connecting to PostgreSQL database")
	db, err := database.NewDB(cfg.PostgresDSN)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		slog.Info("Tip: Start Postgres with 'docker compose up -d postgres' or ensure Postgres is running")
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("Successfully connected to PostgreSQL database")

	// Initialize Kafka consumer
	slog.Info("Connecting to Kafka consumer", "topic", cfg.NotificationsReadyTopic)
	kafkaConsumer, err := consumer.NewConsumer(cfg.KafkaBrokers, cfg.NotificationsReadyTopic, cfg.ConsumerGroupID)
	if err != nil {
		slog.Error("Failed to create Kafka consumer", "error", err)
		slog.Info("Tip: Start Kafka with 'docker compose up -d kafka'")
		os.Exit(1)
	}
	defer kafkaConsumer.Close()
	slog.Info("Successfully connected to Kafka consumer")

	// Initialize sender coordinator (supports email, Slack, and webhook)
	notifSender := sender.NewSender()
	slog.Info("Initialized notification sender coordinator")

	// Main processing loop
	slog.Info("Starting notification sending loop")
	if err := processNotifications(ctx, kafkaConsumer, db, notifSender); err != nil {
		slog.Error("Notification processing failed", "error", err)
		os.Exit(1)
	}

	slog.Info("Sender service stopped")
}

// processNotifications continuously reads notification ready events from Kafka,
// fetches the notification and endpoints from the database, sends notifications via all channels, and updates status.
func processNotifications(ctx context.Context, consumer *consumer.Consumer, db *database.DB, notifSender *sender.Sender) error {
	slog.Info("Starting notification processing loop")

	for {
		select {
		case <-ctx.Done():
			slog.Info("Notification processing loop stopped")
			return nil
		default:
			// Read notification ready event from Kafka
			ready, msg, err := consumer.ReadMessage(ctx)
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
				if err := consumer.CommitMessage(ctx, msg); err != nil {
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
			if err := consumer.CommitMessage(ctx, msg); err != nil {
				slog.Error("Failed to commit offset",
					"notification_id", ready.NotificationID,
					"error", err,
				)
				// Continue processing - offset will be committed on next interval or retry
			}
		}
	}
}

// maskDSN masks sensitive information in the DSN for logging.
func maskDSN(dsn string) string {
	// Simple masking: replace password with ***
	// This is a basic implementation - in production, use a proper DSN parser
	if len(dsn) > 50 {
		return dsn[:20] + "***" + dsn[len(dsn)-20:]
	}
	return "***"
}
