// Package main provides the CLI entry point for the rule-service.
// It handles command-line flag parsing, service initialization, and HTTP server setup.
package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rule-service/internal/config"
	"rule-service/internal/database"
	"rule-service/internal/handlers"
	"rule-service/internal/producer"
	"rule-service/internal/router"

	"github.com/afikmenashe/alerting-platform/pkg/metrics"
)

func main() {
	// Parse command-line flags with environment variable fallbacks
	cfg := &config.Config{}
	flag.StringVar(&cfg.HTTPPort, "http-port", metrics.GetEnvOrDefault("HTTP_PORT", "8081"), "HTTP server port")
	flag.StringVar(&cfg.KafkaBrokers, "kafka-brokers", metrics.GetEnvOrDefault("KAFKA_BROKERS", "localhost:9092"), "Kafka broker addresses (comma-separated)")
	flag.StringVar(&cfg.RuleChangedTopic, "rule-changed-topic", metrics.GetEnvOrDefault("RULE_CHANGED_TOPIC", "rule.changed"), "Kafka topic for rule changed events")
	flag.StringVar(&cfg.PostgresDSN, "postgres-dsn", metrics.GetEnvOrDefault("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable"), "PostgreSQL connection string")
	flag.StringVar(&cfg.RedisAddr, "redis-addr", metrics.GetEnvOrDefault("REDIS_ADDR", "localhost:6379"), "Redis server address")
	flag.Parse()

	// Set up structured logging
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	slog.Info("Starting rule-service",
		"http_port", cfg.HTTPPort,
		"kafka_brokers", cfg.KafkaBrokers,
		"rule_changed_topic", cfg.RuleChangedTopic,
		"postgres_dsn", metrics.MaskDSN(cfg.PostgresDSN),
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
	redisClient, err := metrics.ConnectRedis(ctx, cfg.RedisAddr)
	if err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
		slog.Info("Tip: Start Redis with 'docker compose up -d redis'")
		os.Exit(1)
	}
	defer redisClient.Close()
	slog.Info("Successfully connected to Redis")

	// Initialize metrics reader (for reading other services' metrics)
	metricsReader := metrics.NewReader(redisClient)

	// Initialize metrics collector (for this service's own metrics)
	metricsCollector := metrics.NewCollector("rule-service", redisClient)
	metricsCollector.Start(ctx)
	defer metricsCollector.Stop()

	// Initialize Kafka producer
	slog.Info("Connecting to Kafka producer", "topic", cfg.RuleChangedTopic)
	kafkaProducer, err := producer.NewProducer(cfg.KafkaBrokers, cfg.RuleChangedTopic)
	if err != nil {
		slog.Error("Failed to create Kafka producer", "error", err)
		slog.Info("Tip: Start Kafka with 'docker compose up -d kafka'")
		os.Exit(1)
	}
	defer kafkaProducer.Close()
	slog.Info("Successfully connected to Kafka producer")

	// Initialize HTTP handlers with metrics
	h := handlers.NewHandlers(db, kafkaProducer, metricsReader, metricsCollector)

	// Create HTTP server with router
	server := router.NewServer(cfg.HTTPPort, h)

	// Start HTTP server in a goroutine
	serverErrChan := make(chan error, 1)
	go func() {
		slog.Info("Starting HTTP server", "port", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrChan <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case <-ctx.Done():
		slog.Info("Shutting down HTTP server...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("Error shutting down server", "error", err)
		}
		slog.Info("HTTP server stopped")
	case err := <-serverErrChan:
		slog.Error("HTTP server error", "error", err)
		os.Exit(1)
	}

	slog.Info("Rule-service stopped")
}

