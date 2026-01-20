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

// scanRule scans a rule from a sql.Row or sql.Rows into a Rule struct.
// Used by GetRule, ListRules, GetRulesUpdatedSince, UpdateRule, ToggleRuleEnabled, and CreateRule.
func scanRule(scanner interface {
	Scan(dest ...interface{}) error
}) (*Rule, error) {
	var rule Rule
	err := scanner.Scan(
		&rule.RuleID,
		&rule.ClientID,
		&rule.Severity,
		&rule.Source,
		&rule.Name,
		&rule.Enabled,
		&rule.Version,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// checkRuleVersionMismatch checks if a rule exists but has a version mismatch.
// Returns an error if the rule exists but version doesn't match, nil otherwise.
func (db *DB) checkRuleVersionMismatch(ctx context.Context, ruleID string, expectedVersion int) error {
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM rules WHERE rule_id = $1)`
	if err := db.conn.QueryRowContext(ctx, checkQuery, ruleID).Scan(&exists); err == nil && exists {
		return fmt.Errorf("rule version mismatch: expected version %d", expectedVersion)
	}
	return nil
}
