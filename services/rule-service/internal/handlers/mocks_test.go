// Package handlers provides test mocks for handler dependencies.
package handlers

import (
	"context"
	"time"

	"rule-service/internal/database"
	"rule-service/internal/events"
)

// mockRepository implements Repository interface for testing.
type mockRepository struct {
	// Callbacks for each method (set these to control behavior)
	CreateClientFn        func(ctx context.Context, clientID, name string) error
	GetClientFn           func(ctx context.Context, clientID string) (*database.Client, error)
	ListClientsFn         func(ctx context.Context, limit, offset int) (*database.ClientListResult, error)
	CreateRuleFn          func(ctx context.Context, clientID, severity, source, name string) (*database.Rule, error)
	GetRuleFn             func(ctx context.Context, ruleID string) (*database.Rule, error)
	ListRulesFn           func(ctx context.Context, clientID *string, limit, offset int) (*database.RuleListResult, error)
	UpdateRuleFn          func(ctx context.Context, ruleID string, severity, source, name string, expectedVersion int) (*database.Rule, error)
	ToggleRuleEnabledFn   func(ctx context.Context, ruleID string, enabled bool, expectedVersion int) (*database.Rule, error)
	DeleteRuleFn          func(ctx context.Context, ruleID string) error
	GetRulesUpdatedSinceFn func(ctx context.Context, since time.Time) ([]*database.Rule, error)
	CreateEndpointFn      func(ctx context.Context, ruleID, endpointType, value string) (*database.Endpoint, error)
	GetEndpointFn         func(ctx context.Context, endpointID string) (*database.Endpoint, error)
	ListEndpointsFn       func(ctx context.Context, ruleID *string, limit, offset int) (*database.EndpointListResult, error)
	UpdateEndpointFn      func(ctx context.Context, endpointID, endpointType, value string) (*database.Endpoint, error)
	ToggleEndpointEnabledFn func(ctx context.Context, endpointID string, enabled bool) (*database.Endpoint, error)
	DeleteEndpointFn      func(ctx context.Context, endpointID string) error
	GetNotificationFn     func(ctx context.Context, notificationID string) (*database.Notification, error)
	ListNotificationsFn   func(ctx context.Context, clientID *string, status *string, limit, offset int) (*database.NotificationListResult, error)
}

func (m *mockRepository) CreateClient(ctx context.Context, clientID, name string) error {
	if m.CreateClientFn != nil {
		return m.CreateClientFn(ctx, clientID, name)
	}
	return nil
}

func (m *mockRepository) GetClient(ctx context.Context, clientID string) (*database.Client, error) {
	if m.GetClientFn != nil {
		return m.GetClientFn(ctx, clientID)
	}
	return &database.Client{ClientID: clientID, Name: "Test"}, nil
}

func (m *mockRepository) ListClients(ctx context.Context, limit, offset int) (*database.ClientListResult, error) {
	if m.ListClientsFn != nil {
		return m.ListClientsFn(ctx, limit, offset)
	}
	return &database.ClientListResult{Clients: []*database.Client{}, Total: 0, Limit: limit, Offset: offset}, nil
}

func (m *mockRepository) CreateRule(ctx context.Context, clientID, severity, source, name string) (*database.Rule, error) {
	if m.CreateRuleFn != nil {
		return m.CreateRuleFn(ctx, clientID, severity, source, name)
	}
	return &database.Rule{RuleID: "rule-1", ClientID: clientID, Severity: severity, Source: source, Name: name, Enabled: true, Version: 1}, nil
}

func (m *mockRepository) GetRule(ctx context.Context, ruleID string) (*database.Rule, error) {
	if m.GetRuleFn != nil {
		return m.GetRuleFn(ctx, ruleID)
	}
	return &database.Rule{RuleID: ruleID, ClientID: "client-1", Severity: "HIGH", Source: "source-1", Name: "alert-1", Enabled: true, Version: 1}, nil
}

func (m *mockRepository) ListRules(ctx context.Context, clientID *string, limit, offset int) (*database.RuleListResult, error) {
	if m.ListRulesFn != nil {
		return m.ListRulesFn(ctx, clientID, limit, offset)
	}
	return &database.RuleListResult{Rules: []*database.Rule{}, Total: 0, Limit: limit, Offset: offset}, nil
}

func (m *mockRepository) UpdateRule(ctx context.Context, ruleID string, severity, source, name string, expectedVersion int) (*database.Rule, error) {
	if m.UpdateRuleFn != nil {
		return m.UpdateRuleFn(ctx, ruleID, severity, source, name, expectedVersion)
	}
	return &database.Rule{RuleID: ruleID, Severity: severity, Source: source, Name: name, Version: expectedVersion + 1}, nil
}

func (m *mockRepository) ToggleRuleEnabled(ctx context.Context, ruleID string, enabled bool, expectedVersion int) (*database.Rule, error) {
	if m.ToggleRuleEnabledFn != nil {
		return m.ToggleRuleEnabledFn(ctx, ruleID, enabled, expectedVersion)
	}
	return &database.Rule{RuleID: ruleID, Enabled: enabled, Version: expectedVersion + 1}, nil
}

func (m *mockRepository) DeleteRule(ctx context.Context, ruleID string) error {
	if m.DeleteRuleFn != nil {
		return m.DeleteRuleFn(ctx, ruleID)
	}
	return nil
}

func (m *mockRepository) GetRulesUpdatedSince(ctx context.Context, since time.Time) ([]*database.Rule, error) {
	if m.GetRulesUpdatedSinceFn != nil {
		return m.GetRulesUpdatedSinceFn(ctx, since)
	}
	return []*database.Rule{}, nil
}

func (m *mockRepository) CreateEndpoint(ctx context.Context, ruleID, endpointType, value string) (*database.Endpoint, error) {
	if m.CreateEndpointFn != nil {
		return m.CreateEndpointFn(ctx, ruleID, endpointType, value)
	}
	return &database.Endpoint{EndpointID: "endpoint-1", RuleID: ruleID, Type: endpointType, Value: value, Enabled: true}, nil
}

func (m *mockRepository) GetEndpoint(ctx context.Context, endpointID string) (*database.Endpoint, error) {
	if m.GetEndpointFn != nil {
		return m.GetEndpointFn(ctx, endpointID)
	}
	return &database.Endpoint{EndpointID: endpointID, RuleID: "rule-1", Type: "email", Value: "test@example.com", Enabled: true}, nil
}

func (m *mockRepository) ListEndpoints(ctx context.Context, ruleID *string, limit, offset int) (*database.EndpointListResult, error) {
	if m.ListEndpointsFn != nil {
		return m.ListEndpointsFn(ctx, ruleID, limit, offset)
	}
	return &database.EndpointListResult{Endpoints: []*database.Endpoint{}, Total: 0, Limit: limit, Offset: offset}, nil
}

func (m *mockRepository) UpdateEndpoint(ctx context.Context, endpointID, endpointType, value string) (*database.Endpoint, error) {
	if m.UpdateEndpointFn != nil {
		return m.UpdateEndpointFn(ctx, endpointID, endpointType, value)
	}
	return &database.Endpoint{EndpointID: endpointID, Type: endpointType, Value: value}, nil
}

func (m *mockRepository) ToggleEndpointEnabled(ctx context.Context, endpointID string, enabled bool) (*database.Endpoint, error) {
	if m.ToggleEndpointEnabledFn != nil {
		return m.ToggleEndpointEnabledFn(ctx, endpointID, enabled)
	}
	return &database.Endpoint{EndpointID: endpointID, Enabled: enabled}, nil
}

func (m *mockRepository) DeleteEndpoint(ctx context.Context, endpointID string) error {
	if m.DeleteEndpointFn != nil {
		return m.DeleteEndpointFn(ctx, endpointID)
	}
	return nil
}

func (m *mockRepository) GetNotification(ctx context.Context, notificationID string) (*database.Notification, error) {
	if m.GetNotificationFn != nil {
		return m.GetNotificationFn(ctx, notificationID)
	}
	return &database.Notification{NotificationID: notificationID, ClientID: "client-1", Status: "RECEIVED"}, nil
}

func (m *mockRepository) ListNotifications(ctx context.Context, clientID *string, status *string, limit, offset int) (*database.NotificationListResult, error) {
	if m.ListNotificationsFn != nil {
		return m.ListNotificationsFn(ctx, clientID, status, limit, offset)
	}
	return &database.NotificationListResult{Notifications: []*database.Notification{}, Total: 0, Limit: limit, Offset: offset}, nil
}

func (m *mockRepository) Close() error {
	return nil
}

// mockPublisher implements RulePublisher interface for testing.
type mockPublisher struct {
	PublishFn  func(ctx context.Context, changed *events.RuleChanged) error
	Published  []*events.RuleChanged // Records all published events
}

func (m *mockPublisher) Publish(ctx context.Context, changed *events.RuleChanged) error {
	m.Published = append(m.Published, changed)
	if m.PublishFn != nil {
		return m.PublishFn(ctx, changed)
	}
	return nil
}

func (m *mockPublisher) Close() error {
	return nil
}

// mockMetrics implements MetricsRecorder interface for testing.
type mockMetrics struct {
	ReceivedCount  int
	ProcessedCount int
	PublishedCount int
	ErrorCount     int
	CustomCounts   map[string]int
}

func (m *mockMetrics) RecordReceived()                { m.ReceivedCount++ }
func (m *mockMetrics) RecordProcessed(_ time.Duration) { m.ProcessedCount++ }
func (m *mockMetrics) RecordPublished()               { m.PublishedCount++ }
func (m *mockMetrics) RecordError()                   { m.ErrorCount++ }
func (m *mockMetrics) IncrementCustom(name string) {
	if m.CustomCounts == nil {
		m.CustomCounts = make(map[string]int)
	}
	m.CustomCounts[name]++
}
