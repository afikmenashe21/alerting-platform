// Package webhook provides webhook notification sending via HTTP POST.
package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"sender/internal/database"
	"sender/internal/sender/payload"
	"sender/internal/sender/validation"
)

// Sender implements webhook notification sending via HTTP POST.
type Sender struct {
	httpClient *http.Client
}

// NewSender creates a new webhook sender.
func NewSender() *Sender {
	return &Sender{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Type returns the endpoint type this sender handles.
func (s *Sender) Type() string {
	return "webhook"
}

var dummyWebhookHosts = []string{
	"example.com",
	"example.org",
	"example.net",
	"test.com",
	"localhost",
	"invalid",
}

func isDummyWebhookURL(endpointValue string) bool {
	parsed, err := url.Parse(endpointValue)
	if err != nil {
		return false
	}

	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return false
	}

	for _, dummy := range dummyWebhookHosts {
		if host == dummy || strings.HasSuffix(host, "."+dummy) {
			return true
		}
	}

	return false
}

// Send sends a notification to a webhook endpoint via HTTP POST.
// The endpointValue should be a webhook URL.
func (s *Sender) Send(ctx context.Context, endpointValue string, notification *database.Notification) error {
	if endpointValue == "" {
		return fmt.Errorf("webhook URL is required")
	}

	// Validate that it's a URL (starts with http:// or https://)
	if !validation.IsValidURL(endpointValue) {
		return fmt.Errorf("invalid webhook URL: %q (must be a valid HTTP/HTTPS URL)", endpointValue)
	}

	if isDummyWebhookURL(endpointValue) {
		slog.Info("Skipping dummy webhook endpoint",
			"webhook_url", endpointValue,
			"notification_id", notification.NotificationID,
		)
		return nil
	}

	// Build webhook payload
	webhookPayload := payload.BuildWebhookPayload(notification)

	// Marshal to JSON
	jsonData, err := json.Marshal(webhookPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", endpointValue, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		slog.Error("Failed to send webhook notification",
			"error", err,
			"webhook_url", endpointValue,
			"notification_id", notification.NotificationID,
		)
		return fmt.Errorf("failed to send webhook notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slog.Error("Webhook returned error status",
			"status_code", resp.StatusCode,
			"webhook_url", endpointValue,
			"notification_id", notification.NotificationID,
		)
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	slog.Info("Successfully sent webhook notification",
		"webhook_url", endpointValue,
		"notification_id", notification.NotificationID,
		"alert_id", notification.AlertID,
		"client_id", notification.ClientID,
	)

	return nil
}
