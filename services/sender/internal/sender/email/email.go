// Package email provides email notification sending via SMTP.
package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"

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
	msg := s.buildEmailMessage(actualFrom, recipients, emailPayload.Subject, emailPayload.Body)

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

// buildEmailMessage builds a complete email message in RFC 822 format.
func (s *Sender) buildEmailMessage(from string, to []string, subject, body string) []byte {
	var msg bytes.Buffer
	now := time.Now().Format(time.RFC1123Z)
	
	// Required headers for proper email format
	msg.WriteString(fmt.Sprintf("From: %s\r\n", from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ", ")))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString(fmt.Sprintf("Date: %s\r\n", now))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	msg.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)
	return msg.Bytes()
}

// sendWithTLS sends an email using TLS/STARTTLS for secure SMTP connections (Gmail, etc.)
func (s *Sender) sendWithTLS(addr string, port int, fromAddr string, recipients []string, msg []byte) error {
	var client *smtp.Client
	var err error

	// For port 465, use TLS from the start (SSL/TLS)
	if port == 465 {
		// Connect with TLS
		conn, err := tls.Dial("tcp", addr, &tls.Config{
			ServerName:         s.smtpHost,
			InsecureSkipVerify: false,
		})
		if err != nil {
			return fmt.Errorf("failed to connect to SMTP server with TLS: %w", err)
		}
		defer conn.Close()

		// Create SMTP client over TLS connection
		client, err = smtp.NewClient(conn, s.smtpHost)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
		defer client.Close()
	} else {
		// For port 587, use STARTTLS
		// Connect to SMTP server
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to connect to SMTP server: %w", err)
		}
		defer conn.Close()

		// Create SMTP client
		client, err = smtp.NewClient(conn, s.smtpHost)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
		defer client.Close()

		// Check if server supports STARTTLS
		if ok, _ := client.Extension("STARTTLS"); ok {
			tlsConfig := &tls.Config{
				ServerName:         s.smtpHost,
				InsecureSkipVerify: false,
			}
			if err := client.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("failed to start TLS: %w", err)
			}
		}
	}

	// Authenticate if credentials provided
	if s.smtpUser != "" && s.smtpPassword != "" {
		slog.Debug("Authenticating with SMTP server", "user", s.smtpUser, "host", s.smtpHost)
		auth := smtp.PlainAuth("", s.smtpUser, s.smtpPassword, s.smtpHost)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
		slog.Debug("SMTP authentication successful")
	}

	// Set sender (fromAddr is already adjusted for Gmail in Send function)
	slog.Debug("Setting sender", "from", fromAddr)
	if err := client.Mail(fromAddr); err != nil {
		return fmt.Errorf("failed to set sender %s: %w (Gmail requires FROM to match authenticated user)", fromAddr, err)
	}
	slog.Debug("Sender set successfully")

	// Set recipients
	for _, recipient := range recipients {
		slog.Debug("Adding recipient", "to", recipient)
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
		slog.Debug("Recipient added successfully", "to", recipient)
	}

	// Send email data
	slog.Debug("Opening DATA command")
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to open data writer: %w", err)
	}
	
	slog.Debug("Writing email data", "size", len(msg))
	if _, err := writer.Write(msg); err != nil {
		writer.Close()
		return fmt.Errorf("failed to write email data: %w", err)
	}
	
	slog.Debug("Closing data writer")
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w (this may indicate Gmail rejected the email)", err)
	}
	slog.Debug("Email data sent successfully")

	// Quit
	slog.Debug("Sending QUIT command")
	if err := client.Quit(); err != nil {
		// Quit errors are usually not critical, but log them
		slog.Warn("Error during SMTP QUIT", "error", err)
	} else {
		slog.Debug("SMTP session closed successfully")
	}

	return nil
}

// parseRecipients parses a comma-separated list of email addresses.
func parseRecipients(value string) []string {
	parts := strings.Split(value, ",")
	recipients := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			recipients = append(recipients, trimmed)
		}
	}
	return recipients
}
