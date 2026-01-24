// Package email provides email notification sending with multiple provider support.
// Uses the Strategy pattern to support SES, Resend, and other email providers.
package email

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"sender/internal/database"
	"sender/internal/sender/email/provider"
	"sender/internal/sender/payload"
)

// Sender implements email notification sending using configurable providers.
type Sender struct {
	registry *provider.Registry
	from     string
}

// NewSender creates a new email sender with all providers registered.
// The provider is selected based on EMAIL_PROVIDER env var (default: auto-detect)
// Priority: resend > ses (based on configuration availability)
func NewSender() *Sender {
	from := getEnvOrDefault("EMAIL_FROM", getEnvOrDefault("SES_FROM", "alerts@alerting-platform.local"))

	// Create provider registry
	registry := provider.NewRegistry()

	// Register all providers
	registry.Register(provider.NewSESProvider())
	registry.Register(provider.NewResendProvider())

	// Set provider priority based on EMAIL_PROVIDER env var
	primaryProvider := getEnvOrDefault("EMAIL_PROVIDER", "")

	if primaryProvider != "" {
		// Explicit provider selection
		if err := registry.SetPrimary(primaryProvider); err != nil {
			slog.Warn("Failed to set primary email provider", "provider", primaryProvider, "error", err)
		}
	} else {
		// Auto-detect: prefer Resend if configured, otherwise SES
		if p, ok := registry.Get("resend"); ok && p.IsConfigured() {
			registry.SetPrimary("resend")
		} else {
			registry.SetPrimary("ses")
		}
	}

	// Set fallback order
	registry.SetFallback("resend", "ses")

	// Log active provider
	if p, err := registry.GetPrimary(); err == nil {
		slog.Info("Email sender initialized",
			"primary_provider", p.Name(),
			"from", from,
			"available_providers", registry.List(),
		)
	} else {
		slog.Warn("No email provider configured", "error", err)
	}

	return &Sender{
		registry: registry,
		from:     from,
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// Type returns the endpoint type this sender handles.
func (s *Sender) Type() string {
	return "email"
}

// Send sends an email notification using the configured provider.
func (s *Sender) Send(ctx context.Context, endpointValue string, notification *database.Notification) error {
	if endpointValue == "" {
		return fmt.Errorf("email recipient is required")
	}

	recipients := parseRecipients(endpointValue)
	if len(recipients) == 0 {
		return fmt.Errorf("no valid email recipients provided")
	}

	for _, recipient := range recipients {
		if !strings.Contains(recipient, "@") {
			return fmt.Errorf("invalid email address format: %q (missing @ symbol)", recipient)
		}
	}

	// Build email payload
	emailPayload := payload.BuildEmailPayload(notification)

	// Create provider request
	req := &provider.EmailRequest{
		From:    s.from,
		To:      recipients,
		Subject: emailPayload.Subject,
		Body:    emailPayload.Body,
		HTML:    emailPayload.HTML,
	}

	// Send via registry (handles provider selection and fallback)
	if err := s.registry.Send(ctx, req); err != nil {
		slog.Error("Failed to send email",
			"error", err,
			"to", strings.Join(recipients, ", "),
			"notification_id", notification.NotificationID,
		)
		return err
	}

	slog.Info("Successfully sent email",
		"from", s.from,
		"to", strings.Join(recipients, ", "),
		"subject", emailPayload.Subject,
		"notification_id", notification.NotificationID,
		"alert_id", notification.AlertID,
		"client_id", notification.ClientID,
	)

	return nil
}

// GetActiveProvider returns the name of the currently active email provider.
func (s *Sender) GetActiveProvider() string {
	if p, err := s.registry.GetPrimary(); err == nil {
		return p.Name()
	}
	return "none"
}

// parseRecipients splits a comma-separated list of email addresses.
func parseRecipients(value string) []string {
	parts := strings.Split(value, ",")
	var recipients []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			recipients = append(recipients, trimmed)
		}
	}
	return recipients
}
