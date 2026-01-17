// Package database provides database operations for querying rules.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/lib/pq"
)

// Rule represents a rule record in the database.
type Rule struct {
	RuleID    string
	ClientID  string
	Severity  string
	Source    string
	Name      string
	Enabled   bool
	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DB wraps a database connection and provides rule operations.
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

// GetAllEnabledRules retrieves all enabled rules from the database.
// This is used to rebuild the complete snapshot.
func (db *DB) GetAllEnabledRules(ctx context.Context) ([]*Rule, error) {
	query := `
		SELECT rule_id, client_id, severity, source, name, enabled, version, created_at, updated_at
		FROM rules
		WHERE enabled = TRUE
		ORDER BY created_at ASC
	`
	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query enabled rules: %w", err)
	}
	defer rows.Close()

	var rules []*Rule
	for rows.Next() {
		var rule Rule
		if err := rows.Scan(
			&rule.RuleID,
			&rule.ClientID,
			&rule.Severity,
			&rule.Source,
			&rule.Name,
			&rule.Enabled,
			&rule.Version,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}
		rules = append(rules, &rule)
	}
	return rules, rows.Err()
}

// GetRule retrieves a rule by ID from the database.
// This is used to fetch rule details for incremental updates.
func (db *DB) GetRule(ctx context.Context, ruleID string) (*Rule, error) {
	query := `
		SELECT rule_id, client_id, severity, source, name, enabled, version, created_at, updated_at
		FROM rules
		WHERE rule_id = $1
	`
	var rule Rule
	err := db.conn.QueryRowContext(ctx, query, ruleID).Scan(
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
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("rule not found: %s", ruleID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}
	return &rule, nil
}
