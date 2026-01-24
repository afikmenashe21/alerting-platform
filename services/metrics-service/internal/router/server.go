// Package router provides HTTP routing configuration for the metrics-service API.
package router

import (
	"net/http"
	"time"

	"metrics-service/internal/handlers"
)

// NewServer creates a new HTTP server with the router configured.
func NewServer(port string, h *handlers.Handlers) *http.Server {
	router := NewRouter(h)
	return &http.Server{
		Addr:         ":" + port,
		Handler:      router.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}
