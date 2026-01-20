// Package email provides email notification sending via SMTP.
package email

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

// buildEmailMessage builds a complete email message in RFC 822 format.
func buildEmailMessage(from string, to []string, subject, body string) []byte {
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
