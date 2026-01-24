// Package router provides HTTP routing configuration for the metrics-service API.
package router

import (
	"net/http"

	"metrics-service/internal/handlers"
)

// Router wraps the HTTP mux and provides route configuration.
type Router struct {
	mux      *http.ServeMux
	handlers *handlers.Handlers
}

// NewRouter creates a new router with all routes configured.
func NewRouter(h *handlers.Handlers) *Router {
	r := &Router{
		mux:      http.NewServeMux(),
		handlers: h,
	}
	r.setupRoutes()
	return r
}

// Handler returns the HTTP handler with CORS and metrics middleware applied.
func (r *Router) Handler() http.Handler {
	handler := corsMiddleware(r.mux)
	handler = metricsMiddleware(r.handlers.GetMetricsCollector())(handler)
	return handler
}
