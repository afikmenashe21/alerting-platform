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

	"github.com/afikmenashe/alerting-platform/pkg/metrics"
	"github.com/afikmenashe/alerting-platform/pkg/shared"
)

func main() {
	// Parse command-line flags with environment variable fallbacks
	cfg := &config.Config{}
	flag.StringVar(&cfg.KafkaBrokers, "kafka-brokers", shared.GetEnvOrDefault("KAFKA_BROKERS", "localhost:9092"), "Kafka broker addresses (comma-separated)")
	flag.StringVar(&cfg.NotificationsReadyTopic, "notifications-ready-topic", shared.GetEnvOrDefault("NOTIFICATIONS_READY_TOPIC", "notifications.ready"), "Kafka topic for ready notifications")
	flag.StringVar(&cfg.ConsumerGroupID, "consumer-group-id", shared.GetEnvOrDefault("CONSUMER_GROUP_ID", "sender-group"), "Kafka consumer group ID")
	flag.StringVar(&cfg.PostgresDSN, "postgres-dsn", shared.GetEnvOrDefault("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable"), "PostgreSQL connection string")
	flag.StringVar(&cfg.RedisAddr, "redis-addr", shared.GetEnvOrDefault("REDIS_ADDR", "localhost:6379"), "Redis server address")
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
		"postgres_dsn", shared.MaskDSN(cfg.PostgresDSN),
		"redis_addr", cfg.RedisAddr,
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

	// Initialize Redis client for metrics
	slog.Info("Connecting to Redis", "addr", cfg.RedisAddr)
	redisClient, err := shared.ConnectRedis(ctx, cfg.RedisAddr)
	if err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
		slog.Info("Tip: Start Redis with 'docker compose up -d redis'")
		os.Exit(1)
	}
	defer redisClient.Close()
	slog.Info("Successfully connected to Redis")

	// Initialize metrics collector
	metricsCollector := metrics.NewCollector("sender", redisClient)
	metricsCollector.Start(ctx)
	defer metricsCollector.Stop()

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
	if err := processNotifications(ctx, kafkaConsumer, db, notifSender, metricsCollector); err != nil {
		slog.Error("Notification processing failed", "error", err)
		os.Exit(1)
	}

	slog.Info("Sender service stopped")
}

