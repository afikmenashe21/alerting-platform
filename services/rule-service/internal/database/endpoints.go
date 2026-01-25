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

// ListEndpoints retrieves endpoints with pagination, optionally filtered by rule_id.
// Default limit is 50, max limit is 200.
func (db *DB) ListEndpoints(ctx context.Context, ruleID *string, limit, offset int) (*EndpointListResult, error) {
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
	whereClause := ""
	var countArgs []interface{}
	argIndex := 1

	if ruleID != nil {
		whereClause = fmt.Sprintf("WHERE rule_id = $%d", argIndex)
		countArgs = append(countArgs, *ruleID)
		argIndex++
	}

	// Get total count - use cached count for exact result with fast response
	var total int64
	if ruleID == nil {
		// Unfiltered: use counts cache for exact count (updated by triggers)
		cacheQuery := `SELECT row_count FROM table_counts WHERE table_name = 'endpoints'`
		if err := db.conn.QueryRowContext(ctx, cacheQuery).Scan(&total); err != nil {
			// Fallback to COUNT(*) if cache not available
			countQuery := "SELECT COUNT(*) FROM endpoints"
			_ = db.conn.QueryRowContext(ctx, countQuery).Scan(&total)
		}
	} else {
		// Filtered: use exact count (filter reduces scan significantly)
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM endpoints %s", whereClause)
		if err := db.conn.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
			return nil, fmt.Errorf("failed to count endpoints: %w", err)
		}
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT endpoint_id, rule_id, type, value, enabled, created_at, updated_at
		FROM endpoints
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args := append(countArgs, limit, offset)
	rows, err := db.conn.QueryContext(ctx, query, args...)
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

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &EndpointListResult{
		Endpoints: endpoints,
		Total:     total,
		Limit:     limit,
		Offset:    offset,
	}, nil
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
