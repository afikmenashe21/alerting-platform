// Package database provides database operations for clients, rules, and endpoints.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// CreateRule creates a new rule in the database.
// Returns the created rule with generated rule_id and version.
func (db *DB) CreateRule(ctx context.Context, clientID, severity, source, name string) (*Rule, error) {
	query := `
		INSERT INTO rules (client_id, severity, source, name, enabled, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, TRUE, 1, NOW(), NOW())
		RETURNING rule_id, client_id, severity, source, name, enabled, version, created_at, updated_at
	`
	var rule Rule
	err := db.conn.QueryRowContext(ctx, query, clientID, severity, source, name).Scan(
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
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // unique_violation
				// For wildcard rules, we allow multiple rules with same pattern
				// Only exact matches are prevented by unique constraint
				if severity != "*" && source != "*" && name != "*" {
					return nil, fmt.Errorf("rule already exists for client %s with criteria (severity=%s, source=%s, name=%s)", clientID, severity, source, name)
				}
				// If it's a wildcard rule, the constraint might still fail if exact duplicate
				// This is acceptable - user can have multiple wildcard rules
			}
			if pqErr.Code == "23503" { // foreign_key_violation
				return nil, fmt.Errorf("client not found: %s", clientID)
			}
		}
		return nil, fmt.Errorf("failed to create rule: %w", err)
	}
	return &rule, nil
}

// GetRule retrieves a rule by ID.
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

// ListRules retrieves all rules, optionally filtered by client_id.
func (db *DB) ListRules(ctx context.Context, clientID *string) ([]*Rule, error) {
	var query string
	var args []interface{}

	if clientID != nil {
		query = `
			SELECT rule_id, client_id, severity, source, name, enabled, version, created_at, updated_at
			FROM rules
			WHERE client_id = $1
			ORDER BY created_at DESC
		`
		args = []interface{}{*clientID}
	} else {
		query = `
			SELECT rule_id, client_id, severity, source, name, enabled, version, created_at, updated_at
			FROM rules
			ORDER BY created_at DESC
		`
		args = []interface{}{}
	}

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list rules: %w", err)
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

// UpdateRule updates a rule with optimistic locking.
// Returns the updated rule or an error if version mismatch.
func (db *DB) UpdateRule(ctx context.Context, ruleID string, severity, source, name string, expectedVersion int) (*Rule, error) {
	query := `
		UPDATE rules
		SET severity = $2,
		    source = $3,
		    name = $4,
		    version = version + 1,
		    updated_at = NOW()
		WHERE rule_id = $1 AND version = $5
		RETURNING rule_id, client_id, severity, source, name, enabled, version, created_at, updated_at
	`
	var rule Rule
	err := db.conn.QueryRowContext(ctx, query, ruleID, severity, source, name, expectedVersion).Scan(
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
		// Check if rule exists but version mismatch
		var exists bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM rules WHERE rule_id = $1)`
		if err := db.conn.QueryRowContext(ctx, checkQuery, ruleID).Scan(&exists); err == nil && exists {
			return nil, fmt.Errorf("rule version mismatch: expected version %d", expectedVersion)
		}
		return nil, fmt.Errorf("rule not found: %s", ruleID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update rule: %w", err)
	}
	return &rule, nil
}

// ToggleRuleEnabled toggles the enabled status of a rule with optimistic locking.
func (db *DB) ToggleRuleEnabled(ctx context.Context, ruleID string, enabled bool, expectedVersion int) (*Rule, error) {
	query := `
		UPDATE rules
		SET enabled = $2,
		    version = version + 1,
		    updated_at = NOW()
		WHERE rule_id = $1 AND version = $3
		RETURNING rule_id, client_id, severity, source, name, enabled, version, created_at, updated_at
	`
	var rule Rule
	err := db.conn.QueryRowContext(ctx, query, ruleID, enabled, expectedVersion).Scan(
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
		var exists bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM rules WHERE rule_id = $1)`
		if err := db.conn.QueryRowContext(ctx, checkQuery, ruleID).Scan(&exists); err == nil && exists {
			return nil, fmt.Errorf("rule version mismatch: expected version %d", expectedVersion)
		}
		return nil, fmt.Errorf("rule not found: %s", ruleID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to toggle rule enabled: %w", err)
	}
	return &rule, nil
}

// DeleteRule deletes a rule by ID.
func (db *DB) DeleteRule(ctx context.Context, ruleID string) error {
	query := `DELETE FROM rules WHERE rule_id = $1`
	result, err := db.conn.ExecContext(ctx, query, ruleID)
	if err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("rule not found: %s", ruleID)
	}
	return nil
}

// GetRulesUpdatedSince retrieves rules updated after a given timestamp.
func (db *DB) GetRulesUpdatedSince(ctx context.Context, since time.Time) ([]*Rule, error) {
	query := `
		SELECT rule_id, client_id, severity, source, name, enabled, version, created_at, updated_at
		FROM rules
		WHERE updated_at > $1
		ORDER BY updated_at ASC
	`
	rows, err := db.conn.QueryContext(ctx, query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules updated since: %w", err)
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
