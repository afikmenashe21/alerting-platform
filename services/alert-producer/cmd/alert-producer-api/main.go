// Package main provides the HTTP API server for alert-producer.
// It exposes REST endpoints for generating alerts with optional manual configuration.
package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"

	"alert-producer/internal/api"
)

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	var (
		port              = flag.String("port", envOrDefault("PORT", "8082"), "HTTP server port")
		defaultKafkaBrokers = flag.String("kafka-brokers", envOrDefault("KAFKA_BROKERS", "localhost:9092"), "Default Kafka broker addresses")
	)
	flag.Parse()

	// Create job manager
	jm := api.NewJobManager()

	// Setup routes
	mux := http.NewServeMux()
	mux.HandleFunc("/health", api.HandleHealth)
	mux.HandleFunc("/api/v1/alerts/generate", api.HandleGenerate(jm, *defaultKafkaBrokers))
	mux.HandleFunc("/api/v1/alerts/generate/list", api.HandleListJobs(jm))
	mux.HandleFunc("/api/v1/alerts/generate/status", api.HandleGetJob(jm))
	mux.HandleFunc("/api/v1/alerts/generate/stop", api.HandleStopJob(jm))

	// CORS middleware
	handler := corsMiddleware(mux)

	addr := ":" + *port
	slog.Info("Starting alert-producer API server",
		"port", *port,
		"kafka_brokers", *defaultKafkaBrokers,
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
