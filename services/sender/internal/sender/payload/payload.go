// Package payload provides payload builders for different notification channels.
package payload

import (
	"fmt"
	"strings"
	"time"

	"sender/internal/database"
)

// EmailPayload represents email message content.
type EmailPayload struct {
	Subject string
	Body    string
}

// BuildEmailPayload builds email subject and body from a notification.
func BuildEmailPayload(notification *database.Notification) EmailPayload {
	subject := fmt.Sprintf("Alert: %s - %s", notification.Severity, notification.Name)
	body := buildEmailBody(notification)
	return EmailPayload{
		Subject: subject,
		Body:    body,
	}
}

// buildEmailBody builds the email body from the notification.
func buildEmailBody(notification *database.Notification) string {
	var sb strings.Builder
	sb.WriteString("Alert Notification\n")
	sb.WriteString("==================\n\n")
	sb.WriteString(fmt.Sprintf("Severity: %s\n", notification.Severity))
	sb.WriteString(fmt.Sprintf("Source: %s\n", notification.Source))
	sb.WriteString(fmt.Sprintf("Name: %s\n", notification.Name))
	sb.WriteString(fmt.Sprintf("Alert ID: %s\n", notification.AlertID))
	sb.WriteString(fmt.Sprintf("Client ID: %s\n", notification.ClientID))
	sb.WriteString(fmt.Sprintf("Notification ID: %s\n", notification.NotificationID))
	sb.WriteString(fmt.Sprintf("Matched Rule IDs: %s\n", strings.Join(notification.RuleIDs, ", ")))

	if len(notification.Context) > 0 {
		sb.WriteString("\nContext:\n")
		for k, v := range notification.Context {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	}

	return sb.String()
}

// SlackPayload represents a Slack webhook payload.
type SlackPayload struct {
	Text        string       `json:"text,omitempty"`
	Attachments []Attachment  `json:"attachments,omitempty"`
}

// Attachment represents a Slack message attachment.
type Attachment struct {
	Color     string  `json:"color,omitempty"`
	Title     string  `json:"title,omitempty"`
	Text      string  `json:"text,omitempty"`
	Fields    []Field `json:"fields,omitempty"`
	Timestamp int64   `json:"ts,omitempty"`
}

// Field represents a field in a Slack attachment.
type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// BuildSlackPayload builds a Slack webhook payload from the notification.
func BuildSlackPayload(notification *database.Notification) SlackPayload {
	// Determine color based on severity
	color := getSeverityColor(notification.Severity)

	// Build fields
	fields := []Field{
		{Title: "Severity", Value: notification.Severity, Short: true},
		{Title: "Source", Value: notification.Source, Short: true},
		{Title: "Name", Value: notification.Name, Short: true},
		{Title: "Alert ID", Value: notification.AlertID, Short: true},
		{Title: "Client ID", Value: notification.ClientID, Short: true},
		{Title: "Notification ID", Value: notification.NotificationID, Short: true},
	}

	if len(notification.RuleIDs) > 0 {
		fields = append(fields, Field{
			Title: "Matched Rule IDs",
			Value: strings.Join(notification.RuleIDs, ", "),
			Short: false,
		})
	}

	// Build attachment text
	var text strings.Builder
	text.WriteString(fmt.Sprintf("*Alert: %s*\n", notification.Name))
	if len(notification.Context) > 0 {
		text.WriteString("\n*Context:*\n")
		for k, v := range notification.Context {
			text.WriteString(fmt.Sprintf("• %s: %s\n", k, v))
		}
	}

	return SlackPayload{
		Attachments: []Attachment{
			{
				Color:  color,
				Title:  fmt.Sprintf("Alert: %s - %s", notification.Severity, notification.Name),
				Text:   text.String(),
				Fields: fields,
			},
		},
	}
}

// getSeverityColor returns the Slack color for a given severity.
func getSeverityColor(severity string) string {
	switch strings.ToUpper(severity) {
	case "CRITICAL":
		return "danger" // red
	case "HIGH", "MEDIUM":
		return "warning" // yellow
	case "LOW":
		return "good" // green
	default:
		return "good" // default to green
	}
}

// WebhookPayload represents a webhook payload.
type WebhookPayload struct {
	NotificationID string            `json:"notification_id"`
	ClientID       string            `json:"client_id"`
	AlertID        string            `json:"alert_id"`
	Severity       string            `json:"severity"`
	Source         string            `json:"source"`
	Name           string            `json:"name"`
	Context        map[string]string `json:"context,omitempty"`
	RuleIDs        []string          `json:"rule_ids"`
	Timestamp      string            `json:"timestamp"`
}

// BuildWebhookPayload builds a webhook payload from the notification.
func BuildWebhookPayload(notification *database.Notification) WebhookPayload {
	return WebhookPayload{
		NotificationID: notification.NotificationID,
		ClientID:       notification.ClientID,
		AlertID:        notification.AlertID,
		Severity:       notification.Severity,
		Source:         notification.Source,
		Name:           notification.Name,
		Context:        notification.Context,
		RuleIDs:        notification.RuleIDs,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
	}
}
