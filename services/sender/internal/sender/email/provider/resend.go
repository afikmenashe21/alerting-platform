// Package provider provides email provider implementations.
package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/resend/resend-go/v2"
)

// ResendProvider implements email sending via Resend API.
type ResendProvider struct {
	client *resend.Client
	apiKey string
}

// NewResendProvider creates a new Resend email provider.
// API key is read from RESEND_API_KEY environment variable.
func NewResendProvider() *ResendProvider {
	apiKey := GetEnvOrDefault("RESEND_API_KEY", "")

	if apiKey == "" {
		slog.Warn("RESEND_API_KEY not set, Resend provider will be unavailable")
		return &ResendProvider{}
	}

	client := resend.NewClient(apiKey)
	slog.Info("Resend email provider initialized")

	return &ResendProvider{
		client: client,
		apiKey: apiKey,
	}
}

// Name returns the provider name.
func (p *ResendProvider) Name() string {
	return "resend"
}

// IsConfigured returns true if Resend is properly configured.
func (p *ResendProvider) IsConfigured() bool {
	return p.client != nil && p.apiKey != ""
}

// Send sends an email via Resend API.
func (p *ResendProvider) Send(ctx context.Context, req *EmailRequest) error {
	if p.client == nil {
		return fmt.Errorf("Resend client not initialized")
	}

	if len(req.To) == 0 {
		return fmt.Errorf("no recipients specified")
	}

	// Build Resend request
	params := &resend.SendEmailRequest{
		From:    req.From,
		To:      req.To,
		Subject: req.Subject,
	}

	// Prefer HTML if available, otherwise use plain text
	if req.HTML != "" {
		params.Html = req.HTML
	} else if req.Body != "" {
		params.Text = req.Body
	}

	result, err := p.client.Emails.Send(params)
	if err != nil {
		slog.Error("Resend send failed",
			"error", err,
			"to", req.To,
			"subject", req.Subject,
		)
		return fmt.Errorf("Resend send failed: %w", err)
	}

	slog.Info("Email sent via Resend",
		"email_id", result.Id,
		"to", req.To,
		"subject", req.Subject,
	)

	return nil
}
