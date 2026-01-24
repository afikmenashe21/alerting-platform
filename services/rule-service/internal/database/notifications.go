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

// NotificationListResult contains paginated notification results.
type NotificationListResult struct {
	Notifications []*Notification `json:"notifications"`
	Total         int64           `json:"total"`
	Limit         int             `json:"limit"`
	Offset        int             `json:"offset"`
}

// ListNotifications retrieves notifications with pagination, optionally filtered by client_id or status.
// Default limit is 50, max limit is 200.
func (db *DB) ListNotifications(ctx context.Context, clientID *string, status *string, limit, offset int) (*NotificationListResult, error) {
	// Apply default and max limits
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	// Build WHERE clause
	var whereClauses []string
	var args []interface{}
	argIndex := 1

	if clientID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("client_id = $%d", argIndex))
		args = append(args, *clientID)
		argIndex++
	}
	if status != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *status)
		argIndex++
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + whereClauses[0]
		for i := 1; i < len(whereClauses); i++ {
			whereClause += " AND " + whereClauses[i]
		}
	}

	// Get total count - use approximate count for unfiltered queries (instant for large tables)
	var total int64
	if len(whereClauses) == 0 {
		// Unfiltered: use pg_stat for instant approximate count
		approxQuery := `SELECT n_live_tup FROM pg_stat_user_tables WHERE relname = 'notifications'`
		if err := db.conn.QueryRowContext(ctx, approxQuery).Scan(&total); err != nil {
			// Fallback to regular count if pg_stat fails
			countQuery := "SELECT COUNT(*) FROM notifications"
			_ = db.conn.QueryRowContext(ctx, countQuery).Scan(&total)
		}
	} else {
		// Filtered: use exact count (filters reduce the scan significantly)
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM notifications %s", whereClause)
		if err := db.conn.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
			return nil, fmt.Errorf("failed to count notifications: %w", err)
		}
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT notification_id, client_id, alert_id, severity, source, name, context, rule_ids, status, created_at, updated_at
		FROM notifications
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)
	args = append(args, limit, offset)

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

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &NotificationListResult{
		Notifications: notifications,
		Total:         total,
		Limit:         limit,
		Offset:        offset,
	}, nil
}
