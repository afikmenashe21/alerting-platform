// Package database provides database operations for notifications and endpoints tables.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

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
