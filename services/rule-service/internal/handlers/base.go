// Package handlers provides HTTP handlers for the rule-service API.
package handlers

import (
	"rule-service/internal/database"
	"rule-service/internal/producer"

	"github.com/afikmenashe/alerting-platform/pkg/metrics"
)

const (
	SchemaVersion = 1
)

// Handlers wraps dependencies for HTTP handlers.
type Handlers struct {
	db               *database.DB
	producer         *producer.Producer
	metricsReader    *metrics.Reader
	metricsCollector *metrics.Collector
}

// NewHandlers creates a new handlers instance.
func NewHandlers(db *database.DB, producer *producer.Producer, metricsReader *metrics.Reader, metricsCollector *metrics.Collector) *Handlers {
	return &Handlers{
		db:               db,
		producer:         producer,
		metricsReader:    metricsReader,
		metricsCollector: metricsCollector,
	}
}

// GetMetricsCollector returns the metrics collector for middleware use.
func (h *Handlers) GetMetricsCollector() *metrics.Collector {
	return h.metricsCollector
}
