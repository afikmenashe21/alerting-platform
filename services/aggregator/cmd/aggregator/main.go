package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"aggregator/internal/config"
	"aggregator/internal/consumer"
	"aggregator/internal/database"
	"aggregator/internal/processor"
	"aggregator/internal/producer"
)

func main() {
	// Parse command-line flags with environment variable fallbacks
	cfg := &config.Config{}
	flag.StringVar(&cfg.KafkaBrokers, "kafka-brokers", getEnvOrDefault("KAFKA_BROKERS", "localhost:9092"), "Kafka broker addresses (comma-separated)")
	flag.StringVar(&cfg.AlertsMatchedTopic, "alerts-matched-topic", getEnvOrDefault("ALERTS_MATCHED_TOPIC", "alerts.matched"), "Kafka topic for matched alerts")
	flag.StringVar(&cfg.NotificationsReadyTopic, "notifications-ready-topic", getEnvOrDefault("NOTIFICATIONS_READY_TOPIC", "notifications.ready"), "Kafka topic for ready notifications")
	flag.StringVar(&cfg.ConsumerGroupID, "consumer-group-id", getEnvOrDefault("CONSUMER_GROUP_ID", "aggregator-group"), "Kafka consumer group ID")
	flag.StringVar(&cfg.PostgresDSN, "postgres-dsn", getEnvOrDefault("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable"), "PostgreSQL connection string")
	flag.Parse()

	// Set up structured logging
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	slog.Info("Starting aggregator service",
		"kafka_brokers", cfg.KafkaBrokers,
		"alerts_matched_topic", cfg.AlertsMatchedTopic,
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
	db, err := database.NewDB(cfg.PostgresDSN)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		slog.Info("Tip: Start Postgres with 'docker compose up -d postgres' or ensure Postgres is running")
		os.Exit(1)
	}
	defer db.Close()

	// Initialize Kafka consumer
	slog.Info("Connecting to Kafka consumer", "topic", cfg.AlertsMatchedTopic)
	kafkaConsumer, err := consumer.NewConsumer(cfg.KafkaBrokers, cfg.AlertsMatchedTopic, cfg.ConsumerGroupID)
	if err != nil {
		slog.Error("Failed to create Kafka consumer", "error", err)
		slog.Info("Tip: Start Kafka with 'docker compose up -d kafka'")
		os.Exit(1)
	}
	defer kafkaConsumer.Close()
	slog.Info("Successfully connected to Kafka consumer")

	// Initialize Kafka producer
	slog.Info("Connecting to Kafka producer", "topic", cfg.NotificationsReadyTopic)
	kafkaProducer, err := producer.NewProducer(cfg.KafkaBrokers, cfg.NotificationsReadyTopic)
	if err != nil {
		slog.Error("Failed to create Kafka producer", "error", err)
		os.Exit(1)
	}
	defer kafkaProducer.Close()
	slog.Info("Successfully connected to Kafka producer")

	// Initialize processor
	proc := processor.NewProcessor(kafkaConsumer, kafkaProducer, db)

	// Main processing loop
	if err := proc.ProcessNotifications(ctx); err != nil {
		slog.Error("Notification processing failed", "error", err)
		os.Exit(1)
	}

	slog.Info("Aggregator service stopped")
}


// getEnvOrDefault returns the environment variable value or a default value if not set.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
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
