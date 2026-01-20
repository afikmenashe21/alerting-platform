// Package database provides database operations for the notifications table.
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

// DB wraps a database connection and provides notification operations.
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

// marshalContextToJSONB serializes a context map to a sql.NullString for JSONB storage.
// Returns a NullString with Valid=false if context is nil or empty (NULL in database).
func marshalContextToJSONB(context map[string]string) (sql.NullString, error) {
	var contextJSON sql.NullString
	if context != nil && len(context) > 0 {
		jsonBytes, err := json.Marshal(context)
		if err != nil {
			return sql.NullString{}, fmt.Errorf("failed to marshal context: %w", err)
		}
		contextJSON = sql.NullString{
			String: string(jsonBytes),
			Valid:  true,
		}
	}
	// If context is nil or empty, contextJSON.Valid is false (NULL in database)
	return contextJSON, nil
}

// InsertNotificationIdempotent inserts a notification with idempotency protection.
// Uses INSERT ... ON CONFLICT DO NOTHING RETURNING to ensure no duplicates.
// Returns the notification_id if a new row was inserted, or nil if it already existed.
func (db *DB) InsertNotificationIdempotent(ctx context.Context, clientID, alertID, severity, source, name string, context map[string]string, ruleIDs []string) (*string, error) {
	// Serialize context map to JSONB
	contextJSON, err := marshalContextToJSONB(context)
	if err != nil {
		return nil, err
	}

	// Use pq.Array to properly handle PostgreSQL array type
	// This ensures proper escaping and formatting
	query := `
		INSERT INTO notifications (client_id, alert_id, severity, source, name, context, rule_ids, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'RECEIVED')
		ON CONFLICT (client_id, alert_id) DO NOTHING
		RETURNING notification_id
	`

	var notificationID string
	err = db.conn.QueryRowContext(ctx, query,
		clientID,
		alertID,
		severity,
		source,
		name,
		contextJSON,
		pq.Array(ruleIDs),
	).Scan(&notificationID)

	if err != nil {
		if err == sql.ErrNoRows {
			// No row was inserted (conflict occurred, row already exists)
			slog.Debug("Notification already exists, skipping",
				"client_id", clientID,
				"alert_id", alertID,
			)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to insert notification: %w", err)
	}

	slog.Info("Inserted new notification",
		"notification_id", notificationID,
		"client_id", clientID,
		"alert_id", alertID,
	)

	return &notificationID, nil
}
