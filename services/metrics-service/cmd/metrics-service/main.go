// Package main provides the CLI entry point for the metrics-service.
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

	"metrics-service/internal/config"
	"metrics-service/internal/database"
	"metrics-service/internal/handlers"
	"metrics-service/internal/router"

	"github.com/afikmenashe/alerting-platform/pkg/metrics"
	"github.com/afikmenashe/alerting-platform/pkg/shared"
)

func main() {
	// Parse command-line flags with environment variable fallbacks
	cfg := &config.Config{}
	flag.StringVar(&cfg.HTTPPort, "http-port", shared.GetEnvOrDefault("HTTP_PORT", "8083"), "HTTP server port")
	flag.StringVar(&cfg.PostgresDSN, "postgres-dsn", shared.GetEnvOrDefault("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable"), "PostgreSQL connection string")
	flag.StringVar(&cfg.RedisAddr, "redis-addr", shared.GetEnvOrDefault("REDIS_ADDR", "localhost:6379"), "Redis server address")
	flag.Parse()

	// Set up structured logging
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	slog.Info("Starting metrics-service",
		"http_port", cfg.HTTPPort,
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

	// Initialize metrics reader (for reading other services' metrics)
	metricsReader := metrics.NewReader(redisClient)

	// Initialize metrics collector (for this service's own metrics)
	metricsCollector := metrics.NewCollector("metrics-service", redisClient)
	metricsCollector.Start(ctx)
	defer metricsCollector.Stop()

	// Initialize HTTP handlers
	h := handlers.NewHandlers(db, metricsReader, metricsCollector)

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

	slog.Info("Metrics-service stopped")
}
