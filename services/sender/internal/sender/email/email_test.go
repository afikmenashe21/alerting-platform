package email

import (
	"context"
	"os"
	"strings"
	"testing"

	"sender/internal/database"
	"sender/internal/sender/email/provider"
)

func TestNewSender(t *testing.T) {
	sender := NewSender()
	if sender == nil {
		t.Fatal("NewSender() returned nil")
	}
	if sender.from == "" {
		t.Error("NewSender() from should not be empty")
	}
	if sender.registry == nil {
		t.Error("NewSender() registry should not be nil")
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	os.Setenv("TEST_ENV_VAR", "test-value")
	defer os.Unsetenv("TEST_ENV_VAR")

	val := getEnvOrDefault("TEST_ENV_VAR", "default")
	if val != "test-value" {
		t.Errorf("getEnvOrDefault() = %v, want test-value", val)
	}

	val = getEnvOrDefault("NON_EXISTENT_VAR", "default-value")
	if val != "default-value" {
		t.Errorf("getEnvOrDefault() = %v, want default-value", val)
	}
}

func TestSender_Type(t *testing.T) {
	sender := NewSender()
	if sender.Type() != "email" {
		t.Errorf("Type() = %v, want email", sender.Type())
	}
}

func TestSender_Send_EmptyRecipient(t *testing.T) {
	sender := &Sender{
		from:     "test@example.com",
		registry: provider.NewRegistry(),
	}
	notification := &database.Notification{NotificationID: "notif-123"}
	err := sender.Send(context.Background(), "", notification)
	if err == nil || !strings.Contains(err.Error(), "email recipient is required") {
		t.Errorf("expected 'email recipient is required' error, got %v", err)
	}
}

func TestSender_Send_InvalidEmail(t *testing.T) {
	sender := &Sender{
		from:     "test@example.com",
		registry: provider.NewRegistry(),
	}
	notification := &database.Notification{NotificationID: "notif-123"}
	err := sender.Send(context.Background(), "invalid-email", notification)
	if err == nil || !strings.Contains(err.Error(), "invalid email address format") {
		t.Errorf("expected 'invalid email address format' error, got %v", err)
	}
}

func TestSender_Send_NoValidRecipients(t *testing.T) {
	sender := &Sender{
		from:     "test@example.com",
		registry: provider.NewRegistry(),
	}
	notification := &database.Notification{NotificationID: "notif-123"}
	err := sender.Send(context.Background(), ", ,", notification)
	if err == nil || !strings.Contains(err.Error(), "no valid email recipients provided") {
		t.Errorf("expected 'no valid email recipients' error, got %v", err)
	}
}

func TestSender_Send_NoProvider(t *testing.T) {
	// Empty registry - no providers configured
	sender := &Sender{
		from:     "test@example.com",
		registry: provider.NewRegistry(),
	}
	notification := &database.Notification{NotificationID: "notif-123"}
	err := sender.Send(context.Background(), "test@example.com", notification)
	if err == nil || !strings.Contains(err.Error(), "no configured email provider") {
		t.Errorf("expected 'no configured email provider' error, got %v", err)
	}
}

func TestSender_GetActiveProvider(t *testing.T) {
	sender := &Sender{
		from:     "test@example.com",
		registry: provider.NewRegistry(),
	}
	// No provider configured
	if sender.GetActiveProvider() != "none" {
		t.Errorf("GetActiveProvider() = %v, want 'none'", sender.GetActiveProvider())
	}
}

func TestParseRecipients(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantCount int
	}{
		{"single email", "test@example.com", 1},
		{"multiple emails", "a@b.com,c@d.com,e@f.com", 3},
		{"emails with spaces", "a@b.com, c@d.com , e@f.com", 3},
		{"empty string", "", 0},
		{"only spaces", " , , ", 0},
		{"trailing comma", "test@example.com,", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recipients := parseRecipients(tt.value)
			if len(recipients) != tt.wantCount {
				t.Errorf("parseRecipients() count = %v, want %v", len(recipients), tt.wantCount)
			}
		})
	}
}
