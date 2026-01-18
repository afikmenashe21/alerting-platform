package payload

import (
	"strings"
	"testing"
	"time"

	"sender/internal/database"
)

func TestBuildEmailPayload(t *testing.T) {
	notification := &database.Notification{
		NotificationID: "notif-123",
		ClientID:       "client-456",
		AlertID:        "alert-789",
		Severity:       "HIGH",
		Source:         "test-source",
		Name:           "Test Alert",
		Context:        map[string]string{"key1": "value1", "key2": "value2"},
		RuleIDs:        []string{"rule-001", "rule-002"},
		Status:         "RECEIVED",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	payload := BuildEmailPayload(notification)

	if payload.Subject == "" {
		t.Error("BuildEmailPayload() subject should not be empty")
	}

	if !strings.Contains(payload.Subject, "HIGH") {
		t.Errorf("BuildEmailPayload() subject should contain severity, got %s", payload.Subject)
	}

	if !strings.Contains(payload.Subject, "Test Alert") {
		t.Errorf("BuildEmailPayload() subject should contain name, got %s", payload.Subject)
	}

	if payload.Body == "" {
		t.Error("BuildEmailPayload() body should not be empty")
	}

	if !strings.Contains(payload.Body, "notif-123") {
		t.Errorf("BuildEmailPayload() body should contain notification ID")
	}

	if !strings.Contains(payload.Body, "client-456") {
		t.Errorf("BuildEmailPayload() body should contain client ID")
	}

	if !strings.Contains(payload.Body, "alert-789") {
		t.Errorf("BuildEmailPayload() body should contain alert ID")
	}

	if !strings.Contains(payload.Body, "rule-001") {
		t.Errorf("BuildEmailPayload() body should contain rule IDs")
	}

	if !strings.Contains(payload.Body, "key1") || !strings.Contains(payload.Body, "value1") {
		t.Errorf("BuildEmailPayload() body should contain context")
	}
}

func TestBuildEmailPayload_NoContext(t *testing.T) {
	notification := &database.Notification{
		NotificationID: "notif-123",
		ClientID:       "client-456",
		AlertID:        "alert-789",
		Severity:       "LOW",
		Source:         "test-source",
		Name:           "Test Alert",
		Context:        map[string]string{},
		RuleIDs:        []string{},
		Status:         "RECEIVED",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	payload := BuildEmailPayload(notification)

	if strings.Contains(payload.Body, "Context:") {
		t.Error("BuildEmailPayload() body should not contain context section when context is empty")
	}
}

func TestBuildSlackPayload(t *testing.T) {
	notification := &database.Notification{
		NotificationID: "notif-123",
		ClientID:       "client-456",
		AlertID:        "alert-789",
		Severity:       "CRITICAL",
		Source:         "test-source",
		Name:           "Test Alert",
		Context:        map[string]string{"key1": "value1"},
		RuleIDs:        []string{"rule-001"},
		Status:         "RECEIVED",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	payload := BuildSlackPayload(notification)

	if len(payload.Attachments) == 0 {
		t.Error("BuildSlackPayload() should have at least one attachment")
	}

	attachment := payload.Attachments[0]

	if attachment.Color == "" {
		t.Error("BuildSlackPayload() attachment should have color")
	}

	if attachment.Title == "" {
		t.Error("BuildSlackPayload() attachment should have title")
	}

	if !strings.Contains(attachment.Title, "CRITICAL") {
		t.Errorf("BuildSlackPayload() title should contain severity")
	}

	if len(attachment.Fields) == 0 {
		t.Error("BuildSlackPayload() attachment should have fields")
	}

	// Check that fields contain expected data
	foundSeverity := false
	for _, field := range attachment.Fields {
		if field.Title == "Severity" && field.Value == "CRITICAL" {
			foundSeverity = true
			break
		}
	}
	if !foundSeverity {
		t.Error("BuildSlackPayload() should have Severity field")
	}
}

func TestBuildSlackPayload_NoRuleIDs(t *testing.T) {
	notification := &database.Notification{
		NotificationID: "notif-123",
		ClientID:       "client-456",
		AlertID:        "alert-789",
		Severity:       "MEDIUM",
		Source:         "test-source",
		Name:           "Test Alert",
		Context:        map[string]string{},
		RuleIDs:        []string{},
		Status:         "RECEIVED",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	payload := BuildSlackPayload(notification)

	attachment := payload.Attachments[0]

	// Should not have Matched Rule IDs field when rule IDs are empty
	for _, field := range attachment.Fields {
		if field.Title == "Matched Rule IDs" {
			t.Error("BuildSlackPayload() should not have Matched Rule IDs field when rule IDs are empty")
		}
	}
}

func TestGetSeverityColor(t *testing.T) {
	tests := []struct {
		name     string
		severity string
		want     string
	}{
		{
			name:     "CRITICAL",
			severity: "CRITICAL",
			want:     "danger",
		},
		{
			name:     "critical lowercase",
			severity: "critical",
			want:     "danger",
		},
		{
			name:     "HIGH",
			severity: "HIGH",
			want:     "warning",
		},
		{
			name:     "MEDIUM",
			severity: "MEDIUM",
			want:     "warning",
		},
		{
			name:     "LOW",
			severity: "LOW",
			want:     "good",
		},
		{
			name:     "unknown severity",
			severity: "UNKNOWN",
			want:     "good",
		},
		{
			name:     "empty severity",
			severity: "",
			want:     "good",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a notification with the severity
			notification := &database.Notification{
				Severity: tt.severity,
			}
			payload := BuildSlackPayload(notification)
			if len(payload.Attachments) == 0 {
				t.Fatal("BuildSlackPayload() should have at least one attachment")
			}
			got := payload.Attachments[0].Color
			if got != tt.want {
				t.Errorf("getSeverityColor(%s) = %v, want %v", tt.severity, got, tt.want)
			}
		})
	}
}

func TestBuildWebhookPayload(t *testing.T) {
	notification := &database.Notification{
		NotificationID: "notif-123",
		ClientID:       "client-456",
		AlertID:        "alert-789",
		Severity:       "HIGH",
		Source:         "test-source",
		Name:           "Test Alert",
		Context:        map[string]string{"key1": "value1"},
		RuleIDs:        []string{"rule-001", "rule-002"},
		Status:         "RECEIVED",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	payload := BuildWebhookPayload(notification)

	if payload.NotificationID != "notif-123" {
		t.Errorf("BuildWebhookPayload() NotificationID = %v, want notif-123", payload.NotificationID)
	}

	if payload.ClientID != "client-456" {
		t.Errorf("BuildWebhookPayload() ClientID = %v, want client-456", payload.ClientID)
	}

	if payload.AlertID != "alert-789" {
		t.Errorf("BuildWebhookPayload() AlertID = %v, want alert-789", payload.AlertID)
	}

	if payload.Severity != "HIGH" {
		t.Errorf("BuildWebhookPayload() Severity = %v, want HIGH", payload.Severity)
	}

	if payload.Source != "test-source" {
		t.Errorf("BuildWebhookPayload() Source = %v, want test-source", payload.Source)
	}

	if payload.Name != "Test Alert" {
		t.Errorf("BuildWebhookPayload() Name = %v, want Test Alert", payload.Name)
	}

	if len(payload.Context) != 1 {
		t.Errorf("BuildWebhookPayload() Context length = %v, want 1", len(payload.Context))
	}

	if payload.Context["key1"] != "value1" {
		t.Errorf("BuildWebhookPayload() Context[key1] = %v, want value1", payload.Context["key1"])
	}

	if len(payload.RuleIDs) != 2 {
		t.Errorf("BuildWebhookPayload() RuleIDs length = %v, want 2", len(payload.RuleIDs))
	}

	if payload.Timestamp == "" {
		t.Error("BuildWebhookPayload() Timestamp should not be empty")
	}

	// Verify timestamp is valid RFC3339
	_, err := time.Parse(time.RFC3339, payload.Timestamp)
	if err != nil {
		t.Errorf("BuildWebhookPayload() Timestamp is not valid RFC3339: %v", err)
	}
}

func TestBuildWebhookPayload_NoContext(t *testing.T) {
	notification := &database.Notification{
		NotificationID: "notif-123",
		ClientID:       "client-456",
		AlertID:        "alert-789",
		Severity:       "LOW",
		Source:         "test-source",
		Name:           "Test Alert",
		Context:        map[string]string{},
		RuleIDs:        []string{},
		Status:         "RECEIVED",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	payload := BuildWebhookPayload(notification)

	if payload.Context == nil {
		t.Error("BuildWebhookPayload() Context should not be nil")
	}

	if len(payload.Context) != 0 {
		t.Errorf("BuildWebhookPayload() Context should be empty, got %v", payload.Context)
	}
}
