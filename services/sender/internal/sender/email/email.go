// Package email provides email notification sending via SMTP.
package email

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
	"os"
	"strings"
	"sync"

	"sender/internal/database"
	"sender/internal/sender/payload"
)

// Sender implements email notification sending via SMTP.
// It maintains a persistent SMTP connection to avoid TLS handshake overhead per email.
type Sender struct {
	smtpHost     string
	smtpPort     string
	smtpUser     string
	smtpPassword string
	smtpFrom     string
	client       *smtp.Client
	mu           sync.Mutex
}

// Config holds SMTP configuration.
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
}

// NewSender creates a new email sender with default configuration.
func NewSender() *Sender {
	return NewSenderWithConfig(Config{
		Host:     getEnvOrDefault("SMTP_HOST", "localhost"),
		Port:     getEnvOrDefault("SMTP_PORT", "1025"),
		User:     getEnvOrDefault("SMTP_USER", ""),
		Password: getEnvOrDefault("SMTP_PASSWORD", ""),
		From:     getEnvOrDefault("SMTP_FROM", "alerts@alerting-platform.local"),
	})
}

// NewSenderWithConfig creates a new email sender with custom configuration.
func NewSenderWithConfig(cfg Config) *Sender {
	return &Sender{
		smtpHost:     cfg.Host,
		smtpPort:     cfg.Port,
		smtpUser:     cfg.User,
		smtpPassword: cfg.Password,
		smtpFrom:     cfg.From,
	}
}

// getEnvOrDefault gets an environment variable or returns a default value.
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

// Send sends an email notification via SMTP using a persistent connection.
// The endpointValue should be a comma-separated list of email addresses.
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

	emailPayload := payload.BuildEmailPayload(notification)

	actualFrom := s.smtpFrom
	if strings.Contains(s.smtpHost, "gmail.com") && s.smtpUser != "" {
		actualFrom = s.smtpUser
	}

	msg := buildEmailMessage(actualFrom, recipients, emailPayload.Subject, emailPayload.Body)

	// Use persistent SMTP connection (avoids TLS handshake per email)
	if err := s.sendEmail(actualFrom, recipients, msg); err != nil {
		slog.Error("Failed to send email",
			"error", err,
			"smtp_server", fmt.Sprintf("%s:%s", s.smtpHost, s.smtpPort),
			"to", strings.Join(recipients, ", "),
			"notification_id", notification.NotificationID,
		)
		return fmt.Errorf("failed to send email: %w", err)
	}

	slog.Info("Successfully sent email notification",
		"from", actualFrom,
		"to", strings.Join(recipients, ", "),
		"subject", emailPayload.Subject,
		"smtp_server", fmt.Sprintf("%s:%s", s.smtpHost, s.smtpPort),
		"notification_id", notification.NotificationID,
		"alert_id", notification.AlertID,
		"client_id", notification.ClientID,
	)

	return nil
}

// Close closes the persistent SMTP connection.
func (s *Sender) Close() {
	s.closeSMTP()
}
