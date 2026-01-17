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
)

func main() {
	// Parse command-line flags
	cfg := &config.Config{}
	flag.StringVar(&cfg.HTTPPort, "http-port", "8081", "HTTP server port")
	flag.StringVar(&cfg.KafkaBrokers, "kafka-brokers", "localhost:9092", "Kafka broker addresses (comma-separated)")
	flag.StringVar(&cfg.RuleChangedTopic, "rule-changed-topic", "rule.changed", "Kafka topic for rule changed events")
	flag.StringVar(&cfg.PostgresDSN, "postgres-dsn", "postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable", "PostgreSQL connection string")
	flag.Parse()

	// Set up structured logging
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	slog.Info("Starting rule-service",
		"http_port", cfg.HTTPPort,
		"kafka_brokers", cfg.KafkaBrokers,
		"rule_changed_topic", cfg.RuleChangedTopic,
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

	// Initialize HTTP handlers
	h := handlers.NewHandlers(db, kafkaProducer)

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

// maskDSN masks sensitive information in the DSN for logging.
func maskDSN(dsn string) string {
	// Simple masking: replace password with ***
	// This is a basic implementation - in production, use a proper DSN parser
	if len(dsn) > 50 {
		return dsn[:20] + "***" + dsn[len(dsn)-20:]
	}
	return "***"
}
