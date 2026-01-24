// Package main provides the HTTP API server for alert-producer.
// It exposes REST endpoints for generating alerts with optional manual configuration.
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

	"alert-producer/internal/api"

	"github.com/afikmenashe/alerting-platform/pkg/metrics"
	"github.com/afikmenashe/alerting-platform/pkg/shared"
)

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	var (
		port                = flag.String("port", envOrDefault("PORT", "8082"), "HTTP server port")
		defaultKafkaBrokers = flag.String("kafka-brokers", envOrDefault("KAFKA_BROKERS", "localhost:9092"), "Default Kafka broker addresses")
		redisAddr           = flag.String("redis-addr", envOrDefault("REDIS_ADDR", ""), "Redis server address for metrics")
	)
	flag.Parse()

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

	// Initialize Redis client for metrics (optional)
	var metricsCollector *metrics.Collector
	if *redisAddr != "" {
		slog.Info("Connecting to Redis for metrics", "addr", *redisAddr)
		redisClient, err := shared.ConnectRedis(ctx, *redisAddr)
		if err != nil {
			slog.Warn("Failed to connect to Redis, metrics will be disabled", "error", err)
		} else {
			slog.Info("Successfully connected to Redis")
			metricsCollector = metrics.NewCollector("alert-producer", redisClient)
			metricsCollector.Start(ctx)
			defer metricsCollector.Stop()
			defer redisClient.Close()
		}
	}

	// Create job manager
	jm := api.NewJobManager()

	// Setup routes
	mux := http.NewServeMux()
	mux.HandleFunc("/health", api.HandleHealth)
	mux.HandleFunc("/api/v1/alerts/generate", api.HandleGenerate(jm, *defaultKafkaBrokers))
	mux.HandleFunc("/api/v1/alerts/generate/list", api.HandleListJobs(jm))
	mux.HandleFunc("/api/v1/alerts/generate/status", api.HandleGetJob(jm))
	mux.HandleFunc("/api/v1/alerts/generate/stop", api.HandleStopJob(jm))

	// Apply middleware: CORS first, then metrics
	handler := corsMiddleware(mux)
	handler = metricsMiddleware(metricsCollector)(handler)

	addr := ":" + *port
	slog.Info("Starting alert-producer API server",
		"port", *port,
		"kafka_brokers", *defaultKafkaBrokers,
		"redis_addr", *redisAddr,
	)

	if err := http.ListenAndServe(addr, handler); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}

// envOrDefault reads an environment variable or returns a default value.
func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// corsMiddleware adds CORS headers to allow requests from the UI.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// metricsMiddleware tracks HTTP request metrics.
func metricsMiddleware(collector *metrics.Collector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if collector == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Skip health endpoint
			if r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			collector.RecordReceived()
			start := time.Now()

			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(wrapped, r)

			latency := time.Since(start)

			if wrapped.statusCode >= 400 {
				collector.RecordError()
			} else {
				collector.RecordProcessed(latency)
			}

			collector.IncrementCustom("http_" + r.Method)
		})
	}
}
