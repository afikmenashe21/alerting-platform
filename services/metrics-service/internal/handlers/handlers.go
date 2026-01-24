// Package handlers provides HTTP handlers for the metrics-service API.
package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"metrics-service/internal/database"

	"github.com/afikmenashe/alerting-platform/pkg/metrics"
)

// Handlers wraps dependencies for HTTP handlers.
type Handlers struct {
	db               *database.DB
	metricsReader    *metrics.Reader
	metricsCollector *metrics.Collector
}

// NewHandlers creates a new handlers instance.
func NewHandlers(db *database.DB, metricsReader *metrics.Reader, metricsCollector *metrics.Collector) *Handlers {
	return &Handlers{
		db:               db,
		metricsReader:    metricsReader,
		metricsCollector: metricsCollector,
	}
}

// GetMetricsCollector returns the metrics collector for middleware use.
func (h *Handlers) GetMetricsCollector() *metrics.Collector {
	return h.metricsCollector
}

// ServiceMetricsResponse wraps service metrics with known service list.
type ServiceMetricsResponse struct {
	Services      map[string]*metrics.ServiceMetrics `json:"services"`
	KnownServices []string                           `json:"known_services"`
}

// GetSystemMetrics returns aggregated system metrics from the database.
// GET /api/v1/metrics
func (h *Handlers) GetSystemMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	dbMetrics, err := h.db.GetSystemMetrics(ctx)
	if err != nil {
		slog.Error("Failed to get system metrics", "error", err)
		http.Error(w, "Failed to retrieve metrics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(dbMetrics); err != nil {
		slog.Error("Failed to encode metrics response", "error", err)
	}
}

// GetServiceMetrics returns metrics for all services from Redis.
// GET /api/v1/services/metrics
func (h *Handlers) GetServiceMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.metricsReader == nil {
		slog.Error("Metrics reader not configured")
		http.Error(w, "Metrics reader not available", http.StatusInternalServerError)
		return
	}

	// Get specific service if requested
	serviceName := r.URL.Query().Get("service")
	if serviceName != "" {
		serviceMetrics, err := h.metricsReader.GetServiceMetrics(ctx, serviceName)
		if err != nil {
			slog.Warn("Failed to get service metrics", "service", serviceName, "error", err)
			// Return empty metrics with unhealthy status instead of error
			serviceMetrics = &metrics.ServiceMetrics{
				ServiceName: serviceName,
				Status:      "offline",
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(serviceMetrics); err != nil {
			slog.Error("Failed to encode service metrics", "error", err)
		}
		return
	}

	// Get all services
	allMetrics, err := h.metricsReader.GetAllServiceMetrics(ctx)
	if err != nil {
		slog.Error("Failed to get all service metrics", "error", err)
		http.Error(w, "Failed to retrieve service metrics", http.StatusInternalServerError)
		return
	}

	// Include known services that might be offline
	for _, name := range metrics.ServiceNames {
		if _, exists := allMetrics[name]; !exists {
			allMetrics[name] = &metrics.ServiceMetrics{
				ServiceName: name,
				Status:      "offline",
			}
		}
	}

	response := ServiceMetricsResponse{
		Services:      allMetrics,
		KnownServices: metrics.ServiceNames,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode service metrics response", "error", err)
	}
}
