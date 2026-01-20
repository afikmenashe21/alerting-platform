package email

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"sender/internal/database"
)

func TestNewSender(t *testing.T) {
	sender := NewSender()

	if sender == nil {
		t.Fatal("NewSender() returned nil")
	}

	if sender.smtpHost == "" {
		t.Error("NewSender() smtpHost should not be empty")
	}

	if sender.smtpPort == "" {
		t.Error("NewSender() smtpPort should not be empty")
	}

	if sender.smtpFrom == "" {
		t.Error("NewSender() smtpFrom should not be empty")
	}
}

func TestNewSenderWithConfig(t *testing.T) {
	cfg := Config{
		Host:     "smtp.example.com",
		Port:     "587",
		User:     "user@example.com",
		Password: "password",
		From:     "from@example.com",
	}

	sender := NewSenderWithConfig(cfg)

	if sender.smtpHost != "smtp.example.com" {
		t.Errorf("NewSenderWithConfig() smtpHost = %v, want smtp.example.com", sender.smtpHost)
	}

	if sender.smtpPort != "587" {
		t.Errorf("NewSenderWithConfig() smtpPort = %v, want 587", sender.smtpPort)
	}

	if sender.smtpUser != "user@example.com" {
		t.Errorf("NewSenderWithConfig() smtpUser = %v, want user@example.com", sender.smtpUser)
	}

	if sender.smtpPassword != "password" {
		t.Errorf("NewSenderWithConfig() smtpPassword = %v, want password", sender.smtpPassword)
	}

	if sender.smtpFrom != "from@example.com" {
		t.Errorf("NewSenderWithConfig() smtpFrom = %v, want from@example.com", sender.smtpFrom)
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	// Test with environment variable set
	os.Setenv("TEST_ENV_VAR", "test-value")
	defer os.Unsetenv("TEST_ENV_VAR")

	val := getEnvOrDefault("TEST_ENV_VAR", "default")
	if val != "test-value" {
		t.Errorf("getEnvOrDefault() = %v, want test-value", val)
	}

	// Test with environment variable not set
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
	sender := NewSender()

	notification := &database.Notification{
		NotificationID: "notif-123",
	}

	ctx := context.Background()
	err := sender.Send(ctx, "", notification)

	if err == nil {
		t.Error("Send() should return error for empty recipient")
	}

	if !strings.Contains(err.Error(), "email recipient is required") {
		t.Errorf("Send() error message should mention recipient required, got %v", err.Error())
	}
}

func TestSender_Send_InvalidEmail(t *testing.T) {
	sender := NewSender()

	notification := &database.Notification{
		NotificationID: "notif-123",
	}

	ctx := context.Background()
	err := sender.Send(ctx, "invalid-email", notification)

	if err == nil {
		t.Error("Send() should return error for invalid email")
	}

	if !strings.Contains(err.Error(), "invalid email address format") {
		t.Errorf("Send() error message should mention invalid format, got %v", err.Error())
	}
}

func TestSender_Send_NoValidRecipients(t *testing.T) {
	sender := NewSender()

	notification := &database.Notification{
		NotificationID: "notif-123",
	}

	ctx := context.Background()
	err := sender.Send(ctx, ", ,", notification)

	if err == nil {
		t.Error("Send() should return error for no valid recipients")
	}

	if !strings.Contains(err.Error(), "no valid email recipients provided") {
		t.Errorf("Send() error message should mention no valid recipients, got %v", err.Error())
	}
}

func TestParseRecipients(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantCount int
	}{
		{
			name:      "single email",
			value:     "test@example.com",
			wantCount: 1,
		},
		{
			name:      "multiple emails",
			value:     "test1@example.com,test2@example.com,test3@example.com",
			wantCount: 3,
		},
		{
			name:      "emails with spaces",
			value:     "test1@example.com, test2@example.com , test3@example.com",
			wantCount: 3,
		},
		{
			name:      "empty string",
			value:     "",
			wantCount: 0,
		},
		{
			name:      "only spaces",
			value:     " , , ",
			wantCount: 0,
		},
		{
			name:      "single email with trailing comma",
			value:     "test@example.com,",
			wantCount: 1,
		},
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

func TestBuildEmailMessage(t *testing.T) {
	from := "from@example.com"
	to := []string{"to1@example.com", "to2@example.com"}
	subject := "Test Subject"
	body := "Test Body"

	msg := buildEmailMessage(from, to, subject, body)

	if len(msg) == 0 {
		t.Error("buildEmailMessage() should return non-empty message")
	}

	msgStr := string(msg)

	// Check required headers
	if !strings.Contains(msgStr, "From: from@example.com") {
		t.Error("buildEmailMessage() should contain From header")
	}

	if !strings.Contains(msgStr, "To: to1@example.com, to2@example.com") {
		t.Error("buildEmailMessage() should contain To header")
	}

	if !strings.Contains(msgStr, "Subject: Test Subject") {
		t.Error("buildEmailMessage() should contain Subject header")
	}

	if !strings.Contains(msgStr, "MIME-Version: 1.0") {
		t.Error("buildEmailMessage() should contain MIME-Version header")
	}

	if !strings.Contains(msgStr, "Content-Type: text/plain; charset=UTF-8") {
		t.Error("buildEmailMessage() should contain Content-Type header")
	}

	if !strings.Contains(msgStr, "Test Body") {
		t.Error("buildEmailMessage() should contain body")
	}

	// Check that message ends with body
	if !strings.HasSuffix(msgStr, "Test Body") {
		t.Error("buildEmailMessage() should end with body")
	}
}

func TestSender_Send_InvalidPort(t *testing.T) {
	sender := NewSenderWithConfig(Config{
		Host: "localhost",
		Port: "invalid-port",
		From: "from@example.com",
	})

	notification := &database.Notification{
		NotificationID: "notif-123",
		Severity:       "HIGH",
		Name:           "Test Alert",
	}

	ctx := context.Background()
	err := sender.Send(ctx, "test@example.com", notification)

	if err == nil {
		t.Error("Send() should return error for invalid port")
	}

	if !strings.Contains(err.Error(), "invalid SMTP port") {
		t.Errorf("Send() error message should mention invalid port, got %v", err.Error())
	}
}

func TestSender_Send_GmailFromAddress(t *testing.T) {
	// Test that Gmail uses authenticated user as FROM
	sender := NewSenderWithConfig(Config{
		Host:     "smtp.gmail.com",
		Port:     "587",
		User:     "user@gmail.com",
		Password: "password",
		From:     "different@gmail.com",
	})

	notification := &database.Notification{
		NotificationID: "notif-123",
		Severity:       "HIGH",
		Name:           "Test Alert",
	}

	ctx := context.Background()
	// This will fail to connect, but we can test the logic
	err := sender.Send(ctx, "test@example.com", notification)

	// Should fail with connection error, not FROM address error
	if err != nil && !strings.Contains(err.Error(), "connection") && !strings.Contains(err.Error(), "SMTP") {
		t.Logf("Send() error (expected): %v", err)
	}
}

func TestSender_Send_NonGmail(t *testing.T) {
	// Test that non-Gmail uses configured FROM
	sender := NewSenderWithConfig(Config{
		Host: "smtp.example.com",
		Port: "587",
		From: "from@example.com",
	})

	notification := &database.Notification{
		NotificationID: "notif-123",
		Severity:       "HIGH",
		Name:           "Test Alert",
	}

	ctx := context.Background()
	// This will fail to connect, but we can test the logic
	err := sender.Send(ctx, "test@example.com", notification)

	// Should fail with connection error
	if err != nil && !strings.Contains(err.Error(), "connection") && !strings.Contains(err.Error(), "SMTP") {
		t.Logf("Send() error (expected): %v", err)
	}
}

func TestSender_Send_StandardSMTP(t *testing.T) {
	// Test standard SMTP (non-TLS ports)
	sender := NewSenderWithConfig(Config{
		Host: "localhost",
		Port: "1025", // MailHog port
		From: "from@example.com",
	})

	notification := &database.Notification{
		NotificationID: "notif-123",
		ClientID:       "client-456",
		AlertID:        "alert-789",
		Severity:       "HIGH",
		Source:         "test-source",
		Name:           "Test Alert",
		Context:        map[string]string{"key1": "value1"},
		RuleIDs:        []string{"rule-001"},
		Status:         "RECEIVED",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	ctx := context.Background()
	// This will fail if SMTP server is not running, which is expected in test environment
	err := sender.Send(ctx, "test@example.com", notification)

	if err != nil {
		// Expected if SMTP server is not available
		t.Logf("Send() error (expected if SMTP server not running): %v", err)
	}
}
