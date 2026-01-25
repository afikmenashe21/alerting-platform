// Package database provides database operations for clients, rules, and endpoints.
package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

// CreateClient creates a new client in the database.
// Returns an error if the client already exists.
func (db *DB) CreateClient(ctx context.Context, clientID, name string) error {
	query := `
		INSERT INTO clients (client_id, name, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
	`
	_, err := db.conn.ExecContext(ctx, query, clientID, name)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // unique_violation
				return fmt.Errorf("client already exists: %s", clientID)
			}
		}
		return fmt.Errorf("failed to create client: %w", err)
	}
	return nil
}

// GetClient retrieves a client by ID.
func (db *DB) GetClient(ctx context.Context, clientID string) (*Client, error) {
	query := `
		SELECT client_id, name, created_at, updated_at
		FROM clients
		WHERE client_id = $1
	`
	var client Client
	err := db.conn.QueryRowContext(ctx, query, clientID).Scan(
		&client.ClientID,
		&client.Name,
		&client.CreatedAt,
		&client.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("client not found: %s", clientID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}
	return &client, nil
}

// ListClients retrieves clients with pagination.
// Default limit is 50, max limit is 200.
func (db *DB) ListClients(ctx context.Context, limit, offset int) (*ClientListResult, error) {
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

	// Get total count - use cached count for exact result with fast response
	var total int64
	// Try counts cache first (exact count, updated by triggers)
	cacheQuery := `SELECT row_count FROM table_counts WHERE table_name = 'clients'`
	if err := db.conn.QueryRowContext(ctx, cacheQuery).Scan(&total); err != nil {
		// Fallback to COUNT(*) if cache not available
		if err := db.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM clients").Scan(&total); err != nil {
			return nil, fmt.Errorf("failed to count clients: %w", err)
		}
	}

	// Get paginated results
	query := `
		SELECT client_id, name, created_at, updated_at
		FROM clients
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := db.conn.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list clients: %w", err)
	}
	defer rows.Close()

	var clients []*Client
	for rows.Next() {
		var client Client
		if err := rows.Scan(
			&client.ClientID,
			&client.Name,
			&client.CreatedAt,
			&client.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan client: %w", err)
		}
		clients = append(clients, &client)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &ClientListResult{
		Clients: clients,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}, nil
}
