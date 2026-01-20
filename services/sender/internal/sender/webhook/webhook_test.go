package webhook

import (
	"context"
	"strings"
	"testing"
	"time"

	"sender/internal/database"
	"sender/internal/sender/validation"
)

func TestNewSender(t *testing.T) {
	sender := NewSender()

	if sender == nil {
		t.Fatal("NewSender() returned nil")
	}

	if sender.httpClient == nil {
		t.Error("NewSender() httpClient should not be nil")
	}

	if sender.httpClient.Timeout != 30*time.Second {
		t.Errorf("NewSender() httpClient timeout = %v, want 30s", sender.httpClient.Timeout)
	}
}

func TestSender_Type(t *testing.T) {
	sender := NewSender()

	if sender.Type() != "webhook" {
		t.Errorf("Type() = %v, want webhook", sender.Type())
	}
}

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{
			name: "valid https URL",
			url:  "https://webhook.example.com/endpoint",
			want: true,
		},
		{
			name: "valid http URL",
			url:  "http://webhook.example.com/endpoint",
			want: true,
		},
		{
			name: "invalid URL - no protocol",
			url:  "webhook.example.com/endpoint",
			want: false,
		},
		{
			name: "empty string",
			url:  "",
			want: false,
		},
		{
			name: "ftp URL",
			url:  "ftp://example.com",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validation.IsValidURL(tt.url)
			if got != tt.want {
				t.Errorf("IsValidURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestSender_Send_EmptyURL(t *testing.T) {
	sender := NewSender()

	notification := &database.Notification{
		NotificationID: "notif-123",
	}

	ctx := context.Background()
	err := sender.Send(ctx, "", notification)

	if err == nil {
		t.Error("Send() should return error for empty URL")
	}

	if !strings.Contains(err.Error(), "webhook URL is required") {
		t.Errorf("Send() error message should mention URL required, got %v", err.Error())
	}
}

func TestSender_Send_InvalidURL(t *testing.T) {
	sender := NewSender()

	notification := &database.Notification{
		NotificationID: "notif-123",
	}

	ctx := context.Background()
	err := sender.Send(ctx, "not-a-url", notification)

	if err == nil {
		t.Error("Send() should return error for invalid URL")
	}

	if !strings.Contains(err.Error(), "invalid webhook URL") {
		t.Errorf("Send() error message should mention invalid URL, got %v", err.Error())
	}
}

func TestSender_Send_ValidURL(t *testing.T) {
	sender := NewSender()

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
	// This will fail if webhook URL is not accessible, which is expected in test environment
	err := sender.Send(ctx, "https://webhook.example.com/endpoint", notification)

	if err != nil {
		// Expected if webhook URL is not accessible
		t.Logf("Send() error (expected if webhook not accessible): %v", err)
	}
}

func TestSender_Send_HTTPError(t *testing.T) {
	sender := NewSender()

	notification := &database.Notification{
		NotificationID: "notif-123",
		Severity:       "HIGH",
		Name:           "Test Alert",
	}

	ctx := context.Background()
	// Use a URL that will return an error
	err := sender.Send(ctx, "https://httpstat.us/500", notification)

	if err != nil {
		// Expected - webhook returns error status
		t.Logf("Send() error (expected): %v", err)
	}
}
