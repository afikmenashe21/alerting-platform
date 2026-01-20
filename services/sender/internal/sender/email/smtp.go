// Package email provides email notification sending via SMTP.
package email

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
)

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
