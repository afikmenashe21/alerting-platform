// Package email provides email notification sending via AWS SES API.
package email

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"

	"sender/internal/database"
	"sender/internal/sender/payload"
)

// Sender implements email notification sending via AWS SES API.
type Sender struct {
	client *sesv2.Client
	from   string
	region string
}

// NewSender creates a new SES email sender.
func NewSender() *Sender {
	region := getEnvOrDefault("AWS_REGION", "us-east-1")
	from := getEnvOrDefault("SMTP_FROM", getEnvOrDefault("SES_FROM", "alerts@alerting-platform.local"))

	// Load AWS config (uses EC2 instance role credentials automatically)
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		slog.Error("Failed to load AWS config, SES sending will fail", "error", err)
		return &Sender{from: from, region: region}
	}

	client := sesv2.NewFromConfig(cfg)
	slog.Info("SES email sender initialized", "region", region, "from", from)

	return &Sender{
		client: client,
		from:   from,
		region: region,
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

// Send sends an email notification via AWS SES API.
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

	if s.client == nil {
		return fmt.Errorf("SES client not initialized")
	}

	emailPayload := payload.BuildEmailPayload(notification)

	// Build SES SendEmail request
	toAddresses := make([]string, len(recipients))
	copy(toAddresses, recipients)

	input := &sesv2.SendEmailInput{
		FromEmailAddress: &s.from,
		Destination: &types.Destination{
			ToAddresses: toAddresses,
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{
					Data: &emailPayload.Subject,
				},
				Body: &types.Body{
					Text: &types.Content{
						Data: &emailPayload.Body,
					},
				},
			},
		},
	}

	_, err := s.client.SendEmail(ctx, input)
	if err != nil {
		slog.Error("Failed to send email via SES",
			"error", err,
			"to", strings.Join(recipients, ", "),
			"notification_id", notification.NotificationID,
		)
		return fmt.Errorf("SES send failed: %w", err)
	}

	slog.Info("Successfully sent email via SES",
		"from", s.from,
		"to", strings.Join(recipients, ", "),
		"subject", emailPayload.Subject,
		"notification_id", notification.NotificationID,
		"alert_id", notification.AlertID,
		"client_id", notification.ClientID,
	)

	return nil
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
