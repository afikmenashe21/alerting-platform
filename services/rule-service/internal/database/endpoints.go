// Package database provides database operations for clients, rules, and endpoints.
package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

// CreateEndpoint creates a new endpoint for a rule.
func (db *DB) CreateEndpoint(ctx context.Context, ruleID, endpointType, value string) (*Endpoint, error) {
	query := `
		INSERT INTO endpoints (rule_id, type, value, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, TRUE, NOW(), NOW())
		RETURNING endpoint_id, rule_id, type, value, enabled, created_at, updated_at
	`
	var endpoint Endpoint
	err := db.conn.QueryRowContext(ctx, query, ruleID, endpointType, value).Scan(
		&endpoint.EndpointID,
		&endpoint.RuleID,
		&endpoint.Type,
		&endpoint.Value,
		&endpoint.Enabled,
		&endpoint.CreatedAt,
		&endpoint.UpdatedAt,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // unique_violation
				return nil, fmt.Errorf("endpoint already exists for rule %s with type %s and value %s", ruleID, endpointType, value)
			}
			if pqErr.Code == "23503" { // foreign_key_violation
				return nil, fmt.Errorf("rule not found: %s", ruleID)
			}
		}
		return nil, fmt.Errorf("failed to create endpoint: %w", err)
	}
	return &endpoint, nil
}

// GetEndpoint retrieves an endpoint by ID.
func (db *DB) GetEndpoint(ctx context.Context, endpointID string) (*Endpoint, error) {
	query := `
		SELECT endpoint_id, rule_id, type, value, enabled, created_at, updated_at
		FROM endpoints
		WHERE endpoint_id = $1
	`
	var endpoint Endpoint
	err := db.conn.QueryRowContext(ctx, query, endpointID).Scan(
		&endpoint.EndpointID,
		&endpoint.RuleID,
		&endpoint.Type,
		&endpoint.Value,
		&endpoint.Enabled,
		&endpoint.CreatedAt,
		&endpoint.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("endpoint not found: %s", endpointID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint: %w", err)
	}
	return &endpoint, nil
}

// ListEndpoints retrieves all endpoints for a rule.
func (db *DB) ListEndpoints(ctx context.Context, ruleID string) ([]*Endpoint, error) {
	query := `
		SELECT endpoint_id, rule_id, type, value, enabled, created_at, updated_at
		FROM endpoints
		WHERE rule_id = $1
		ORDER BY created_at ASC
	`
	rows, err := db.conn.QueryContext(ctx, query, ruleID)
	if err != nil {
		return nil, fmt.Errorf("failed to list endpoints: %w", err)
	}
	defer rows.Close()

	var endpoints []*Endpoint
	for rows.Next() {
		var endpoint Endpoint
		if err := rows.Scan(
			&endpoint.EndpointID,
			&endpoint.RuleID,
			&endpoint.Type,
			&endpoint.Value,
			&endpoint.Enabled,
			&endpoint.CreatedAt,
			&endpoint.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan endpoint: %w", err)
		}
		endpoints = append(endpoints, &endpoint)
	}
	return endpoints, rows.Err()
}

// UpdateEndpoint updates an endpoint.
func (db *DB) UpdateEndpoint(ctx context.Context, endpointID, endpointType, value string) (*Endpoint, error) {
	query := `
		UPDATE endpoints
		SET type = $2,
		    value = $3,
		    updated_at = NOW()
		WHERE endpoint_id = $1
		RETURNING endpoint_id, rule_id, type, value, enabled, created_at, updated_at
	`
	var endpoint Endpoint
	err := db.conn.QueryRowContext(ctx, query, endpointID, endpointType, value).Scan(
		&endpoint.EndpointID,
		&endpoint.RuleID,
		&endpoint.Type,
		&endpoint.Value,
		&endpoint.Enabled,
		&endpoint.CreatedAt,
		&endpoint.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("endpoint not found: %s", endpointID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update endpoint: %w", err)
	}
	return &endpoint, nil
}

// ToggleEndpointEnabled toggles the enabled status of an endpoint.
func (db *DB) ToggleEndpointEnabled(ctx context.Context, endpointID string, enabled bool) (*Endpoint, error) {
	query := `
		UPDATE endpoints
		SET enabled = $2,
		    updated_at = NOW()
		WHERE endpoint_id = $1
		RETURNING endpoint_id, rule_id, type, value, enabled, created_at, updated_at
	`
	var endpoint Endpoint
	err := db.conn.QueryRowContext(ctx, query, endpointID, enabled).Scan(
		&endpoint.EndpointID,
		&endpoint.RuleID,
		&endpoint.Type,
		&endpoint.Value,
		&endpoint.Enabled,
		&endpoint.CreatedAt,
		&endpoint.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("endpoint not found: %s", endpointID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to toggle endpoint enabled: %w", err)
	}
	return &endpoint, nil
}

// DeleteEndpoint deletes an endpoint by ID.
func (db *DB) DeleteEndpoint(ctx context.Context, endpointID string) error {
	query := `DELETE FROM endpoints WHERE endpoint_id = $1`
	result, err := db.conn.ExecContext(ctx, query, endpointID)
	if err != nil {
		return fmt.Errorf("failed to delete endpoint: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("endpoint not found: %s", endpointID)
	}
	return nil
}
