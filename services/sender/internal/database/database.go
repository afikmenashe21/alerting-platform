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

// Endpoint represents an endpoint record from the endpoints table.
type Endpoint struct {
	EndpointID string
	RuleID     string
	Type       string
	Value      string
	Enabled    bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// DB wraps a database connection and provides notification and endpoint operations.
type DB struct {
	conn *sql.DB
}

// NewDB creates a new database connection using the provided DSN.
func NewDB(dsn string) (*DB, error) {
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	slog.Info("Successfully connected to PostgreSQL database")

	return &DB{conn: conn}, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	if db.conn != nil {
		slog.Info("Closing database connection")
		return db.conn.Close()
	}
	return nil
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

// GetEmailEndpointsByRuleIDs retrieves all enabled email endpoints for the given rule IDs.
// Returns a map of rule_id -> []email addresses.
// Note: ruleIDs may be UUIDs (from actual rules) or test strings (like "rule-001").
// The query casts rule_id to text for comparison, so it works with both.
// DEPRECATED: Use GetEndpointsByRuleIDs instead to get all endpoint types.
func (db *DB) GetEmailEndpointsByRuleIDs(ctx context.Context, ruleIDs []string) (map[string][]string, error) {
	endpoints, err := db.GetEndpointsByRuleIDs(ctx, ruleIDs)
	if err != nil {
		return nil, err
	}

	// Filter to only email endpoints
	result := make(map[string][]string)
	for ruleID, eps := range endpoints {
		for _, ep := range eps {
			if ep.Type == "email" {
				result[ruleID] = append(result[ruleID], ep.Value)
			}
		}
	}
	return result, nil
}

// GetEndpointsByRuleIDs retrieves all enabled endpoints (email, slack, webhook) for the given rule IDs.
// Returns a map of rule_id -> []Endpoint.
// Note: ruleIDs may be UUIDs (from actual rules) or test strings (like "rule-001").
// The query casts rule_id to text for comparison, so it works with both.
func (db *DB) GetEndpointsByRuleIDs(ctx context.Context, ruleIDs []string) (map[string][]Endpoint, error) {
	if len(ruleIDs) == 0 {
		return make(map[string][]Endpoint), nil
	}

	// Filter out empty rule IDs and convert to array for PostgreSQL
	validRuleIDs := make([]string, 0, len(ruleIDs))
	for _, id := range ruleIDs {
		if id != "" {
			validRuleIDs = append(validRuleIDs, id)
		}
	}

	if len(validRuleIDs) == 0 {
		return make(map[string][]Endpoint), nil
	}

	// Cast rule_id::text to compare with TEXT[] elements
	// This handles both UUID rule_ids and test string rule_ids
	query := `
		SELECT rule_id::text, type, value, endpoint_id, enabled, created_at, updated_at
		FROM endpoints
		WHERE rule_id::text = ANY($1) AND enabled = TRUE
		ORDER BY rule_id, created_at ASC
	`

	rows, err := db.conn.QueryContext(ctx, query, pq.Array(validRuleIDs))
	if err != nil {
		return nil, fmt.Errorf("failed to query endpoints: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]Endpoint)
	for rows.Next() {
		var ep Endpoint
		if err := rows.Scan(&ep.RuleID, &ep.Type, &ep.Value, &ep.EndpointID, &ep.Enabled, &ep.CreatedAt, &ep.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan endpoint: %w", err)
		}
		result[ep.RuleID] = append(result[ep.RuleID], ep)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating endpoints: %w", err)
	}

	return result, nil
}
