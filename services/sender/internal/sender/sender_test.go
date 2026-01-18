package sender

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"sender/internal/database"
	"sender/internal/sender/strategy"
)

// mockNotificationSender is a mock implementation of NotificationSender for testing
type mockNotificationSender struct {
	senderType string
	sendErr    error
	sendCalled bool
	endpointValue string
	notification  *database.Notification
}

func (m *mockNotificationSender) Send(ctx context.Context, endpointValue string, notification *database.Notification) error {
	m.sendCalled = true
	m.endpointValue = endpointValue
	m.notification = notification
	return m.sendErr
}

func (m *mockNotificationSender) Type() string {
	return m.senderType
}

func TestNewSender(t *testing.T) {
	s := NewSender()

	if s == nil {
		t.Fatal("NewSender() returned nil")
	}

	if s.registry == nil {
		t.Error("NewSender() registry should not be nil")
	}

	// Verify that default senders are registered
	types := s.registry.List()
	if len(types) == 0 {
		t.Error("NewSender() should register default senders")
	}
}

func TestNewSenderWithRegistry(t *testing.T) {
	registry := strategy.NewRegistry()
	mockSender := &mockNotificationSender{senderType: "test"}
	registry.Register(mockSender)

	s := NewSenderWithRegistry(registry)

	if s == nil {
		t.Fatal("NewSenderWithRegistry() returned nil")
	}

	if s.registry != registry {
		t.Error("NewSenderWithRegistry() should use provided registry")
	}

	// Verify the custom sender is registered
	_, ok := s.registry.Get("test")
	if !ok {
		t.Error("NewSenderWithRegistry() should have custom sender registered")
	}
}

func TestSender_SendNotification(t *testing.T) {
	registry := strategy.NewRegistry()

	emailSender := &mockNotificationSender{senderType: "email"}
	slackSender := &mockNotificationSender{senderType: "slack"}
	webhookSender := &mockNotificationSender{senderType: "webhook"}

	registry.Register(emailSender)
	registry.Register(slackSender)
	registry.Register(webhookSender)

	s := NewSenderWithRegistry(registry)

	notification := &database.Notification{
		NotificationID: "notif-123",
		ClientID:       "client-456",
		AlertID:        "alert-789",
		Severity:       "HIGH",
		Source:         "test-source",
		Name:           "Test Alert",
		Context:        map[string]string{},
		RuleIDs:        []string{"rule-001", "rule-002"},
		Status:         "RECEIVED",
	}

	endpoints := map[string][]database.Endpoint{
		"rule-001": {
			{EndpointID: "ep-001", RuleID: "rule-001", Type: "email", Value: "test1@example.com", Enabled: true},
			{EndpointID: "ep-002", RuleID: "rule-001", Type: "slack", Value: "https://hooks.slack.com/test", Enabled: true},
		},
		"rule-002": {
			{EndpointID: "ep-003", RuleID: "rule-002", Type: "email", Value: "test2@example.com", Enabled: true},
			{EndpointID: "ep-004", RuleID: "rule-002", Type: "webhook", Value: "https://webhook.example.com", Enabled: true},
		},
	}

	ctx := context.Background()
	err := s.SendNotification(ctx, notification, endpoints)

	if err != nil {
		t.Errorf("SendNotification() error = %v, want nil", err)
	}

	// Verify email sender was called
	if !emailSender.sendCalled {
		t.Error("SendNotification() should call email sender")
	}

	// Verify slack sender was called
	if !slackSender.sendCalled {
		t.Error("SendNotification() should call slack sender")
	}

	// Verify webhook sender was called
	if !webhookSender.sendCalled {
		t.Error("SendNotification() should call webhook sender")
	}
}

func TestSender_SendNotification_NoEndpoints(t *testing.T) {
	s := NewSender()

	notification := &database.Notification{
		NotificationID: "notif-123",
		RuleIDs:        []string{"rule-001"},
	}

	endpoints := map[string][]database.Endpoint{}

	ctx := context.Background()
	err := s.SendNotification(ctx, notification, endpoints)

	if err == nil {
		t.Error("SendNotification() should return error when no endpoints")
	}

	if !contains(err.Error(), "no endpoints found") {
		t.Errorf("SendNotification() error message should mention no endpoints, got %v", err.Error())
	}
}

func TestSender_SendNotification_UnknownEndpointType(t *testing.T) {
	registry := strategy.NewRegistry()
	emailSender := &mockNotificationSender{senderType: "email"}
	registry.Register(emailSender)

	s := NewSenderWithRegistry(registry)

	notification := &database.Notification{
		NotificationID: "notif-123",
		RuleIDs:        []string{"rule-001"},
	}

	endpoints := map[string][]database.Endpoint{
		"rule-001": {
			{EndpointID: "ep-001", RuleID: "rule-001", Type: "unknown", Value: "test", Enabled: true},
		},
	}

	ctx := context.Background()
	err := s.SendNotification(ctx, notification, endpoints)

	// Should not return error, but should skip unknown endpoint type
	if err != nil {
		t.Errorf("SendNotification() should not return error for unknown endpoint type, got %v", err)
	}
}

func TestSender_SendNotification_PartialFailure(t *testing.T) {
	registry := strategy.NewRegistry()

	emailSender := &mockNotificationSender{senderType: "email", sendErr: nil}
	slackSender := &mockNotificationSender{senderType: "slack", sendErr: fmt.Errorf("slack error")}

	registry.Register(emailSender)
	registry.Register(slackSender)

	s := NewSenderWithRegistry(registry)

	notification := &database.Notification{
		NotificationID: "notif-123",
		RuleIDs:        []string{"rule-001"},
	}

	endpoints := map[string][]database.Endpoint{
		"rule-001": {
			{EndpointID: "ep-001", RuleID: "rule-001", Type: "email", Value: "test@example.com", Enabled: true},
			{EndpointID: "ep-002", RuleID: "rule-001", Type: "slack", Value: "https://hooks.slack.com/test", Enabled: true},
		},
	}

	ctx := context.Background()
	err := s.SendNotification(ctx, notification, endpoints)

	// Should not return error if at least one send succeeds
	if err != nil {
		t.Errorf("SendNotification() should not return error when at least one send succeeds, got %v", err)
	}
}

func TestSender_SendNotification_AllFailures(t *testing.T) {
	registry := strategy.NewRegistry()

	emailSender := &mockNotificationSender{senderType: "email", sendErr: fmt.Errorf("email error")}
	slackSender := &mockNotificationSender{senderType: "slack", sendErr: fmt.Errorf("slack error")}

	registry.Register(emailSender)
	registry.Register(slackSender)

	s := NewSenderWithRegistry(registry)

	notification := &database.Notification{
		NotificationID: "notif-123",
		RuleIDs:        []string{"rule-001"},
	}

	endpoints := map[string][]database.Endpoint{
		"rule-001": {
			{EndpointID: "ep-001", RuleID: "rule-001", Type: "email", Value: "test@example.com", Enabled: true},
			{EndpointID: "ep-002", RuleID: "rule-001", Type: "slack", Value: "https://hooks.slack.com/test", Enabled: true},
		},
	}

	ctx := context.Background()
	err := s.SendNotification(ctx, notification, endpoints)

	// Should return error if all sends fail
	if err == nil {
		t.Error("SendNotification() should return error when all sends fail")
	}

	if !contains(err.Error(), "all sends failed") {
		t.Errorf("SendNotification() error message should mention all sends failed, got %v", err.Error())
	}
}

func TestSender_groupEndpoints(t *testing.T) {
	s := NewSender()

	endpoints := map[string][]database.Endpoint{
		"rule-001": {
			{EndpointID: "ep-001", RuleID: "rule-001", Type: "email", Value: "test1@example.com", Enabled: true},
			{EndpointID: "ep-002", RuleID: "rule-001", Type: "email", Value: "test2@example.com", Enabled: true},
			{EndpointID: "ep-003", RuleID: "rule-001", Type: "slack", Value: "https://hooks.slack.com/test", Enabled: true},
		},
		"rule-002": {
			{EndpointID: "ep-004", RuleID: "rule-002", Type: "email", Value: "test1@example.com", Enabled: true}, // Duplicate value
			{EndpointID: "ep-005", RuleID: "rule-002", Type: "webhook", Value: "https://webhook.example.com", Enabled: true},
		},
	}

	ruleIDs := []string{"rule-001", "rule-002"}

	grouped := s.groupEndpoints(endpoints, ruleIDs)

	// Check email endpoints (should have unique values)
	if len(grouped["email"]) != 2 {
		t.Errorf("groupEndpoints() email should have 2 unique values, got %d", len(grouped["email"]))
	}

	// Check slack endpoints
	if len(grouped["slack"]) != 1 {
		t.Errorf("groupEndpoints() slack should have 1 value, got %d", len(grouped["slack"]))
	}

	// Check webhook endpoints
	if len(grouped["webhook"]) != 1 {
		t.Errorf("groupEndpoints() webhook should have 1 value, got %d", len(grouped["webhook"]))
	}
}

func TestSender_groupEndpoints_DisabledEndpoints(t *testing.T) {
	s := NewSender()

	endpoints := map[string][]database.Endpoint{
		"rule-001": {
			{EndpointID: "ep-001", RuleID: "rule-001", Type: "email", Value: "test1@example.com", Enabled: true},
			{EndpointID: "ep-002", RuleID: "rule-001", Type: "email", Value: "test2@example.com", Enabled: false},
		},
	}

	ruleIDs := []string{"rule-001"}

	grouped := s.groupEndpoints(endpoints, ruleIDs)

	// Should only include enabled endpoints
	if len(grouped["email"]) != 1 {
		t.Errorf("groupEndpoints() should only include enabled endpoints, got %d", len(grouped["email"]))
	}

	if grouped["email"][0] != "test1@example.com" {
		t.Errorf("groupEndpoints() should include test1@example.com, got %v", grouped["email"])
	}
}

func TestSender_groupEndpoints_EmptyRuleIDs(t *testing.T) {
	s := NewSender()

	endpoints := map[string][]database.Endpoint{
		"rule-001": {
			{EndpointID: "ep-001", RuleID: "rule-001", Type: "email", Value: "test1@example.com", Enabled: true},
		},
	}

	ruleIDs := []string{}

	grouped := s.groupEndpoints(endpoints, ruleIDs)

	if len(grouped) != 0 {
		t.Errorf("groupEndpoints() should return empty map for empty rule IDs, got %v", grouped)
	}
}

func TestSender_groupEndpoints_NonExistentRuleID(t *testing.T) {
	s := NewSender()

	endpoints := map[string][]database.Endpoint{
		"rule-001": {
			{EndpointID: "ep-001", RuleID: "rule-001", Type: "email", Value: "test1@example.com", Enabled: true},
		},
	}

	ruleIDs := []string{"rule-999"}

	grouped := s.groupEndpoints(endpoints, ruleIDs)

	if len(grouped) != 0 {
		t.Errorf("groupEndpoints() should return empty map for non-existent rule ID, got %v", grouped)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
