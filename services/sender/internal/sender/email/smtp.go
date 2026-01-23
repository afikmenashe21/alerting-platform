// Package email provides email notification sending via SMTP.
package email

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
)

// connectSMTP establishes a new authenticated SMTP connection.
func (s *Sender) connectSMTP() (*smtp.Client, error) {
	addr := fmt.Sprintf("%s:%s", s.smtpHost, s.smtpPort)
	var client *smtp.Client

	if s.smtpPort == "465" {
		conn, err := tls.Dial("tcp", addr, &tls.Config{
			ServerName: s.smtpHost,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to connect to SMTP server with TLS: %w", err)
		}
		client, err = smtp.NewClient(conn, s.smtpHost)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to create SMTP client: %w", err)
		}
	} else {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to SMTP server: %w", err)
		}
		client, err = smtp.NewClient(conn, s.smtpHost)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to create SMTP client: %w", err)
		}
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: s.smtpHost}); err != nil {
				client.Close()
				return nil, fmt.Errorf("failed to start TLS: %w", err)
			}
		}
	}

	if s.smtpUser != "" && s.smtpPassword != "" {
		auth := smtp.PlainAuth("", s.smtpUser, s.smtpPassword, s.smtpHost)
		if err := client.Auth(auth); err != nil {
			client.Close()
			return nil, fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	slog.Info("SMTP connection established", "host", s.smtpHost, "port", s.smtpPort)
	return client, nil
}

// getClient returns a connected SMTP client, creating one if needed.
// Must be called with s.mu held.
func (s *Sender) getClient() (*smtp.Client, error) {
	if s.client != nil {
		// Check if connection is still alive with NOOP
		if err := s.client.Noop(); err == nil {
			return s.client, nil
		}
		// Connection dead, close and reconnect
		s.client.Close()
		s.client = nil
	}
	client, err := s.connectSMTP()
	if err != nil {
		return nil, err
	}
	s.client = client
	return client, nil
}

// sendEmail sends an email using the persistent SMTP connection.
// Reconnects automatically if the connection is stale.
func (s *Sender) sendEmail(fromAddr string, recipients []string, msg []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	client, err := s.getClient()
	if err != nil {
		return err
	}

	err = s.sendOnClient(client, fromAddr, recipients, msg)
	if err != nil {
		// Connection might be broken, try once more with fresh connection
		slog.Warn("SMTP send failed, reconnecting", "error", err)
		s.client.Close()
		s.client = nil
		client, err = s.getClient()
		if err != nil {
			return err
		}
		return s.sendOnClient(client, fromAddr, recipients, msg)
	}
	return nil
}

// sendOnClient sends a single email on an existing SMTP client.
func (s *Sender) sendOnClient(client *smtp.Client, fromAddr string, recipients []string, msg []byte) error {
	if err := client.Reset(); err != nil {
		return fmt.Errorf("SMTP RSET failed: %w", err)
	}
	if err := client.Mail(fromAddr); err != nil {
		return fmt.Errorf("failed to set sender %s: %w", fromAddr, err)
	}
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to open data writer: %w", err)
	}
	if _, err := writer.Write(msg); err != nil {
		writer.Close()
		return fmt.Errorf("failed to write email data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}
	return nil
}

// closeSMTP closes the persistent SMTP connection.
func (s *Sender) closeSMTP() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client != nil {
		s.client.Quit()
		s.client = nil
		slog.Info("SMTP connection closed")
	}
}

