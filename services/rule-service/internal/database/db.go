// Package database provides database operations for clients, rules, and endpoints.
package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// DB wraps a database connection and provides client, rule, and endpoint operations.
type DB struct {
	conn *sql.DB
}

// unmarshalNotificationContext deserializes notification context JSON.
func unmarshalNotificationContext(contextJSON sql.NullString, warnAttrs ...any) map[string]string {
	if !contextJSON.Valid || contextJSON.String == "" {
		return make(map[string]string)
	}

	var ctx map[string]string
	if err := json.Unmarshal([]byte(contextJSON.String), &ctx); err != nil {
		slog.Warn("Failed to unmarshal context JSON", append([]any{"error", err}, warnAttrs...)...)
		return make(map[string]string)
	}
	if ctx == nil {
		return make(map[string]string)
	}
	return ctx
}

// NewDB creates a new database connection using the provided DSN.
func NewDB(dsn string) (*DB, error) {
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)

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
