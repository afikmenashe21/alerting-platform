// Package router provides HTTP routing configuration for the rule-service API.
package router

import (
	"net/http"
	"time"

	"github.com/afikmenashe/alerting-platform/pkg/metrics"
)

// corsMiddleware applies CORS headers to all requests.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
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

			// Skip metrics endpoints to avoid recursion
			if r.URL.Path == "/api/v1/services/metrics" || r.URL.Path == "/api/v1/metrics" || r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			collector.RecordReceived()
			start := time.Now()

			// Wrap response writer to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(wrapped, r)

			latency := time.Since(start)

			if wrapped.statusCode >= 400 {
				collector.RecordError()
			} else {
				collector.RecordProcessed(latency)
			}

			// Track by HTTP method
			collector.IncrementCustom("http_" + r.Method)
		})
	}
}
