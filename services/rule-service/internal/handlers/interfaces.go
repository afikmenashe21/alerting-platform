// Package handlers provides HTTP handlers for the rule-service API.
package handlers

import (
	"context"
	"time"

	"rule-service/internal/database"
	"rule-service/internal/events"
)

// RulePublisher defines the interface for publishing rule change events to Kafka.
// This interface allows for dependency injection and easier testing.
type RulePublisher interface {
	// Publish sends a rule changed event to Kafka.
	// Returns an error if serialization or publishing fails.
	Publish(ctx context.Context, changed *events.RuleChanged) error

	// Close gracefully closes the publisher and releases resources.
	Close() error
}

// Repository defines the interface for database operations.
// This allows handlers to be tested without a real database.
type Repository interface {
	// Client operations
	CreateClient(ctx context.Context, clientID, name string) error
	GetClient(ctx context.Context, clientID string) (*database.Client, error)
	ListClients(ctx context.Context, limit, offset int) (*database.ClientListResult, error)

	// Rule operations
	CreateRule(ctx context.Context, clientID, severity, source, name string) (*database.Rule, error)
	GetRule(ctx context.Context, ruleID string) (*database.Rule, error)
	ListRules(ctx context.Context, clientID *string, limit, offset int) (*database.RuleListResult, error)
	UpdateRule(ctx context.Context, ruleID string, severity, source, name string, expectedVersion int) (*database.Rule, error)
	ToggleRuleEnabled(ctx context.Context, ruleID string, enabled bool, expectedVersion int) (*database.Rule, error)
	DeleteRule(ctx context.Context, ruleID string) error
	GetRulesUpdatedSince(ctx context.Context, since time.Time) ([]*database.Rule, error)

	// Endpoint operations
	CreateEndpoint(ctx context.Context, ruleID, endpointType, value string) (*database.Endpoint, error)
	GetEndpoint(ctx context.Context, endpointID string) (*database.Endpoint, error)
	ListEndpoints(ctx context.Context, ruleID *string, limit, offset int) (*database.EndpointListResult, error)
	UpdateEndpoint(ctx context.Context, endpointID, endpointType, value string) (*database.Endpoint, error)
	ToggleEndpointEnabled(ctx context.Context, endpointID string, enabled bool) (*database.Endpoint, error)
	DeleteEndpoint(ctx context.Context, endpointID string) error

	// Notification operations
	GetNotification(ctx context.Context, notificationID string) (*database.Notification, error)
	ListNotifications(ctx context.Context, clientID *string, status *string, limit, offset int) (*database.NotificationListResult, error)

	// Lifecycle
	Close() error
}

// MetricsRecorder defines the interface for recording metrics.
// This uses the null object pattern - a no-op implementation avoids nil checks.
type MetricsRecorder interface {
	RecordReceived()
	RecordProcessed(latency time.Duration)
	RecordPublished()
	RecordError()
	IncrementCustom(name string)
}

// NoOpMetrics is a no-op implementation of MetricsRecorder.
// Use this when metrics collection is not needed, avoiding nil checks.
type NoOpMetrics struct{}

// Ensure NoOpMetrics implements MetricsRecorder.
var _ MetricsRecorder = (*NoOpMetrics)(nil)

func (NoOpMetrics) RecordReceived()                   {}
func (NoOpMetrics) RecordProcessed(_ time.Duration)   {}
func (NoOpMetrics) RecordPublished()                  {}
func (NoOpMetrics) RecordError()                      {}
func (NoOpMetrics) IncrementCustom(_ string)          {}

// metricsAdapter wraps the pkg/metrics.Collector to implement MetricsRecorder.
type metricsAdapter struct {
	collector metricsCollectorInterface
}

// metricsCollectorInterface defines the subset of metrics.Collector methods we use.
type metricsCollectorInterface interface {
	RecordReceived()
	RecordProcessed(latency time.Duration)
	RecordPublished()
	RecordError()
	IncrementCustom(name string)
}

func (a *metricsAdapter) RecordReceived()                 { a.collector.RecordReceived() }
func (a *metricsAdapter) RecordProcessed(d time.Duration) { a.collector.RecordProcessed(d) }
func (a *metricsAdapter) RecordPublished()                { a.collector.RecordPublished() }
func (a *metricsAdapter) RecordError()                    { a.collector.RecordError() }
func (a *metricsAdapter) IncrementCustom(name string)     { a.collector.IncrementCustom(name) }
