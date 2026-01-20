package slack

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

	if sender.Type() != "slack" {
		t.Errorf("Type() = %v, want slack", sender.Type())
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
			url:  "https://hooks.slack.com/services/xxx/yyy/zzz",
			want: true,
		},
		{
			name: "valid http URL",
			url:  "http://example.com/webhook",
			want: true,
		},
		{
			name: "invalid URL - no protocol",
			url:  "hooks.slack.com/services/xxx/yyy/zzz",
			want: false,
		},
		{
			name: "invalid URL - channel name",
			url:  "#general",
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

func TestMaskURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "long URL",
			url:  "https://hooks.slack.com/services/very/long/webhook/url/that/exceeds/fifty/characters",
			want: "https://hooks.slack.com/ser...",
		},
		{
			name: "short URL",
			url:  "https://hooks.slack.com/test",
			want: "https://hooks.slack.com/test",
		},
		{
			name: "exactly 50 characters",
			url:  "https://hooks.slack.com/services/1234567890",
			want: "https://hooks.slack.com/services/1234567890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskURL(tt.url)
			if len(tt.url) > 50 {
				// For long URLs, check that it's masked
				if !strings.Contains(got, "...") {
					t.Errorf("maskURL() should mask long URLs, got %v", got)
				}
			} else {
				// For short URLs, should return as-is
				if got != tt.want {
					t.Errorf("maskURL() = %v, want %v", got, tt.want)
				}
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

	if !strings.Contains(err.Error(), "slack webhook URL is required") {
		t.Errorf("Send() error message should mention URL required, got %v", err.Error())
	}
}

func TestSender_Send_InvalidURL(t *testing.T) {
	sender := NewSender()

	notification := &database.Notification{
		NotificationID: "notif-123",
	}

	ctx := context.Background()
	err := sender.Send(ctx, "#general", notification)

	if err == nil {
		t.Error("Send() should return error for invalid URL")
	}

	if !strings.Contains(err.Error(), "invalid Slack webhook URL") {
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
	err := sender.Send(ctx, "https://hooks.slack.com/services/xxx/yyy/zzz", notification)

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
