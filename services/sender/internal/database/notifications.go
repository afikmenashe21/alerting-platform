// Package database provides database operations for notifications and endpoints tables.
package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/lib/pq"
)

// Notification represents a notification record in the database.
type Notification struct {
	NotificationID string
	ClientID       string
	AlertID        string
	Severity       string
	Source         string
	Name           string
	Context        map[string]string
	RuleIDs        []string
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// GetNotification retrieves a notification by ID.
func (db *DB) GetNotification(ctx context.Context, notificationID string) (*Notification, error) {
	query := `
		SELECT notification_id, client_id, alert_id, severity, source, name, context, rule_ids, status, created_at, updated_at
		FROM notifications
		WHERE notification_id = $1
	`
	var notif Notification
	var contextJSON sql.NullString
	err := db.conn.QueryRowContext(ctx, query, notificationID).Scan(
		&notif.NotificationID,
		&notif.ClientID,
		&notif.AlertID,
		&notif.Severity,
		&notif.Source,
		&notif.Name,
		&contextJSON,
		pq.Array(&notif.RuleIDs),
		&notif.Status,
		&notif.CreatedAt,
		&notif.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("notification not found: %s", notificationID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	// Deserialize context JSON
	if contextJSON.Valid && contextJSON.String != "" {
		if err := json.Unmarshal([]byte(contextJSON.String), &notif.Context); err != nil {
			slog.Warn("Failed to unmarshal context JSON", "error", err, "notification_id", notificationID)
			notif.Context = make(map[string]string)
		}
	} else {
		notif.Context = make(map[string]string)
	}

	return &notif, nil
}

// UpdateNotificationStatus updates the status of a notification.
// This is idempotent: if status is already SENT, it's a no-op.
func (db *DB) UpdateNotificationStatus(ctx context.Context, notificationID string, status string) error {
	query := `
		UPDATE notifications
		SET status = $2, updated_at = NOW()
		WHERE notification_id = $1
	`
	result, err := db.conn.ExecContext(ctx, query, notificationID, status)
	if err != nil {
		return fmt.Errorf("failed to update notification status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("notification not found: %s", notificationID)
	}

	slog.Debug("Updated notification status",
		"notification_id", notificationID,
		"status", status,
	)

	return nil
}
