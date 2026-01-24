// Package provider provides email provider implementations.
package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

// SESProvider implements email sending via AWS SES.
type SESProvider struct {
	client *sesv2.Client
	region string
}

// NewSESProvider creates a new SES email provider.
func NewSESProvider() *SESProvider {
	region := GetEnvOrDefault("AWS_REGION", "us-east-1")

	// Load AWS config (uses EC2 instance role credentials automatically)
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		slog.Warn("Failed to load AWS config, SES provider will be unavailable", "error", err)
		return &SESProvider{region: region}
	}

	client := sesv2.NewFromConfig(cfg)
	slog.Info("SES email provider initialized", "region", region)

	return &SESProvider{
		client: client,
		region: region,
	}
}

// Name returns the provider name.
func (p *SESProvider) Name() string {
	return "ses"
}

// IsConfigured returns true if SES is properly configured.
func (p *SESProvider) IsConfigured() bool {
	return p.client != nil
}

// Send sends an email via AWS SES.
func (p *SESProvider) Send(ctx context.Context, req *EmailRequest) error {
	if p.client == nil {
		return fmt.Errorf("SES client not initialized")
	}

	if len(req.To) == 0 {
		return fmt.Errorf("no recipients specified")
	}

	// Build the email body
	var body types.Body
	if req.HTML != "" {
		body.Html = &types.Content{Data: &req.HTML}
	}
	if req.Body != "" {
		body.Text = &types.Content{Data: &req.Body}
	}

	input := &sesv2.SendEmailInput{
		FromEmailAddress: &req.From,
		Destination: &types.Destination{
			ToAddresses: req.To,
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: &req.Subject},
				Body:    &body,
			},
		},
	}

	result, err := p.client.SendEmail(ctx, input)
	if err != nil {
		slog.Error("SES send failed",
			"error", err,
			"to", req.To,
			"subject", req.Subject,
		)
		return fmt.Errorf("SES send failed: %w", err)
	}

	slog.Info("Email sent via SES",
		"message_id", *result.MessageId,
		"to", req.To,
		"subject", req.Subject,
	)

	return nil
}
