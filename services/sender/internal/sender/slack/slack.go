// Package slack provides Slack notification sending via Incoming Webhooks.
package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"sender/internal/database"
	"sender/internal/sender/payload"
)

// isValidURL checks if a string is a valid HTTP/HTTPS URL.
func isValidURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// maskURL masks sensitive parts of a URL for logging.
func maskURL(url string) string {
	if len(url) > 50 {
		// Show first 30 chars and last 10 chars
		return url[:30] + "..." + url[len(url)-10:]
	}
	return url
}

// Sender implements Slack notification sending via Incoming Webhooks.
type Sender struct {
	httpClient *http.Client
}

// NewSender creates a new Slack sender.
func NewSender() *Sender {
	return &Sender{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Type returns the endpoint type this sender handles.
func (s *Sender) Type() string {
	return "slack"
}

// Send sends a notification to Slack via Incoming Webhook.
// The endpointValue should be a Slack webhook URL.
func (s *Sender) Send(ctx context.Context, endpointValue string, notification *database.Notification) error {
	if endpointValue == "" {
		return fmt.Errorf("slack webhook URL is required")
	}

	// Validate that it's a URL (starts with http:// or https://)
	if !isValidURL(endpointValue) {
		return fmt.Errorf("invalid Slack webhook URL: %q (must be a valid HTTP/HTTPS URL, not a channel name). Slack webhook URLs typically start with https://hooks.slack.com/services/", endpointValue)
	}

	// Build Slack message payload
	slackPayload := payload.BuildSlackPayload(notification)

	// Marshal to JSON
	jsonData, err := json.Marshal(slackPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %w", err)
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
		slog.Error("Failed to send Slack notification",
			"error", err,
			"webhook_url", maskURL(endpointValue),
			"notification_id", notification.NotificationID,
		)
		return fmt.Errorf("failed to send Slack notification to %s: %w", maskURL(endpointValue), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slog.Error("Slack webhook returned error status",
			"status_code", resp.StatusCode,
			"notification_id", notification.NotificationID,
		)
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	slog.Info("Successfully sent Slack notification",
		"notification_id", notification.NotificationID,
		"alert_id", notification.AlertID,
		"client_id", notification.ClientID,
	)

	return nil
}
