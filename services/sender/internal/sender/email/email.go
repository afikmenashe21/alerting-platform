// Package email provides email notification sending via SMTP.
package email

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
	"os"
	"strconv"
	"strings"

	"sender/internal/database"
	"sender/internal/sender/payload"
)

// Sender implements email notification sending via SMTP.
type Sender struct {
	smtpHost     string
	smtpPort     string
	smtpUser     string
	smtpPassword string
	smtpFrom     string
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

// Send sends an email notification via SMTP.
// The endpointValue should be a comma-separated list of email addresses.
func (s *Sender) Send(ctx context.Context, endpointValue string, notification *database.Notification) error {
	if endpointValue == "" {
		return fmt.Errorf("email recipient is required")
	}

	// Parse recipients (comma-separated)
	recipients := parseRecipients(endpointValue)
	if len(recipients) == 0 {
		return fmt.Errorf("no valid email recipients provided")
	}

	// Basic validation: check for @ symbol in email addresses
	for _, recipient := range recipients {
		if !strings.Contains(recipient, "@") {
			return fmt.Errorf("invalid email address format: %q (missing @ symbol)", recipient)
		}
	}

	// Build email content
	emailPayload := payload.BuildEmailPayload(notification)

	// For Gmail, FROM address must match authenticated user
	// We'll use the authenticated user as FROM in the SMTP envelope
	// but can use a different display name in the email headers
	actualFrom := s.smtpFrom
	if strings.Contains(s.smtpHost, "gmail.com") && s.smtpUser != "" {
		// Gmail requires envelope FROM to match authenticated user
		actualFrom = s.smtpUser
		if !strings.EqualFold(s.smtpFrom, s.smtpUser) {
			slog.Info("Gmail: Using authenticated user as FROM address",
				"authenticated_user", s.smtpUser,
				"configured_from", s.smtpFrom,
			)
		}
	}

	// Build email message
	msg := buildEmailMessage(actualFrom, recipients, emailPayload.Subject, emailPayload.Body)

	// Connect to SMTP server
	addr := fmt.Sprintf("%s:%s", s.smtpHost, s.smtpPort)
	port, err := strconv.Atoi(s.smtpPort)
	if err != nil {
		return fmt.Errorf("invalid SMTP port: %s", s.smtpPort)
	}

	// For Gmail and other providers that require TLS, use custom connection
	// Port 587 uses STARTTLS, port 465 uses SSL/TLS
	if port == 587 || port == 465 {
		// Use TLS connection for Gmail and similar providers
		err = s.sendWithTLS(addr, port, actualFrom, recipients, msg)
	} else {
		// Use standard SMTP for local servers (like MailHog)
		var auth smtp.Auth
		if s.smtpUser != "" && s.smtpPassword != "" {
			auth = smtp.PlainAuth("", s.smtpUser, s.smtpPassword, s.smtpHost)
		}
		err = smtp.SendMail(addr, auth, actualFrom, recipients, msg)
	}
	if err != nil {
		slog.Error("Failed to send email",
			"error", err,
			"smtp_server", fmt.Sprintf("%s:%s", s.smtpHost, s.smtpPort),
			"to", strings.Join(recipients, ", "),
			"notification_id", notification.NotificationID,
		)
		
		// Provide helpful error message for connection issues
		if strings.Contains(err.Error(), "connection refused") {
			return fmt.Errorf("failed to send email: %w (SMTP server at %s:%s is not available. Start an SMTP server or configure SMTP_HOST/SMTP_PORT)", err, s.smtpHost, s.smtpPort)
		}
		
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
	
	// Provide helpful note for Gmail
	if strings.Contains(s.smtpHost, "gmail.com") {
		slog.Info("Gmail email sent. Check recipient's inbox and spam folder. Emails may take a few minutes to arrive.")
		slog.Info("If email not in sent folder, Gmail may have rejected it. Check Gmail security activity.")
	}

	return nil
}
