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
	db       Repository
	producer RulePublisher
	metrics  MetricsRecorder
}

// Option is a functional option for configuring Handlers.
type Option func(*Handlers)

// WithMetrics sets a custom metrics recorder.
func WithMetrics(m MetricsRecorder) Option {
	return func(h *Handlers) {
		if m != nil {
			h.metrics = m
		}
	}
}

// NewHandlers creates a new handlers instance.
// If metricsCollector is nil, a no-op implementation is used.
func NewHandlers(db *database.DB, prod *producer.Producer, metricsCollector *metrics.Collector, opts ...Option) *Handlers {
	h := &Handlers{
		db:       db,
		producer: prod,
		metrics:  NoOpMetrics{}, // Default to no-op, never nil
	}

	// If a metrics collector was provided, wrap it
	if metricsCollector != nil {
		h.metrics = &metricsAdapter{collector: metricsCollector}
	}

	// Apply any additional options
	for _, opt := range opts {
		opt(h)
	}

	return h
}

// NewHandlersWithDeps creates handlers with explicit interface dependencies.
// This constructor is primarily for testing.
func NewHandlersWithDeps(db Repository, prod RulePublisher, m MetricsRecorder) *Handlers {
	metrics := m
	if metrics == nil {
		metrics = NoOpMetrics{}
	}
	return &Handlers{
		db:       db,
		producer: prod,
		metrics:  metrics,
	}
}

// GetMetricsCollector returns a metrics.Collector for middleware use.
// Returns nil if the underlying metrics is not a collector.
// This method exists for backward compatibility with the router middleware.
func (h *Handlers) GetMetricsCollector() *metrics.Collector {
	// Check if the metrics adapter wraps a real collector
	if adapter, ok := h.metrics.(*metricsAdapter); ok {
		if collector, ok := adapter.collector.(*metrics.Collector); ok {
			return collector
		}
	}
	return nil
}
