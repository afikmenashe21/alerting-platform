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

// ListClients retrieves all clients.
func (db *DB) ListClients(ctx context.Context) ([]*Client, error) {
	query := `
		SELECT client_id, name, created_at, updated_at
		FROM clients
		ORDER BY created_at DESC
	`
	rows, err := db.conn.QueryContext(ctx, query)
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
	return clients, rows.Err()
}
