// Package router provides HTTP routing configuration for the metrics-service API.
package router

import (
	"net/http"
)

// setupRoutes configures all HTTP routes for the API.
func (r *Router) setupRoutes() {
	// System metrics endpoint (database aggregates)
	r.mux.HandleFunc("/api/v1/metrics", func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodGet {
			r.handlers.GetSystemMetrics(w, req)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Service metrics endpoint (from Redis)
	r.mux.HandleFunc("/api/v1/services/metrics", func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodGet {
			r.handlers.GetServiceMetrics(w, req)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Health check endpoint
	r.mux.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
}
