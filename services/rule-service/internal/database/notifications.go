// Package database provides database operations for clients, rules, and endpoints.
package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

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

	notif.Context = unmarshalNotificationContext(contextJSON, "notification_id", notificationID)

	return &notif, nil
}

// ListNotifications retrieves all notifications, optionally filtered by client_id or status.
func (db *DB) ListNotifications(ctx context.Context, clientID *string, status *string) ([]*Notification, error) {
	var query string
	var args []interface{}

	if clientID != nil && status != nil {
		query = `
			SELECT notification_id, client_id, alert_id, severity, source, name, context, rule_ids, status, created_at, updated_at
			FROM notifications
			WHERE client_id = $1 AND status = $2
			ORDER BY created_at DESC
		`
		args = []interface{}{*clientID, *status}
	} else if clientID != nil {
		query = `
			SELECT notification_id, client_id, alert_id, severity, source, name, context, rule_ids, status, created_at, updated_at
			FROM notifications
			WHERE client_id = $1
			ORDER BY created_at DESC
		`
		args = []interface{}{*clientID}
	} else if status != nil {
		query = `
			SELECT notification_id, client_id, alert_id, severity, source, name, context, rule_ids, status, created_at, updated_at
			FROM notifications
			WHERE status = $1
			ORDER BY created_at DESC
		`
		args = []interface{}{*status}
	} else {
		query = `
			SELECT notification_id, client_id, alert_id, severity, source, name, context, rule_ids, status, created_at, updated_at
			FROM notifications
			ORDER BY created_at DESC
		`
		args = []interface{}{}
	}

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list notifications: %w", err)
	}
	defer rows.Close()

	var notifications []*Notification
	for rows.Next() {
		var notif Notification
		var contextJSON sql.NullString
		if err := rows.Scan(
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
		); err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}

		notif.Context = unmarshalNotificationContext(contextJSON)

		notifications = append(notifications, &notif)
	}
	return notifications, rows.Err()
}
