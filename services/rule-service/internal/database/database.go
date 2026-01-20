// Package database provides database operations for clients, rules, and endpoints.
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

// Client represents a client record in the database.
type Client struct {
	ClientID  string    `json:"client_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Rule represents a rule record in the database.
type Rule struct {
	RuleID    string    `json:"rule_id"`
	ClientID  string    `json:"client_id"`
	Severity  string    `json:"severity"`
	Source    string    `json:"source"`
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Endpoint represents an endpoint record in the database.
type Endpoint struct {
	EndpointID string    `json:"endpoint_id"`
	RuleID     string    `json:"rule_id"`
	Type       string    `json:"type"` // email, webhook, slack
	Value      string    `json:"value"` // email address, URL, etc.
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Notification represents a notification record in the database.
type Notification struct {
	NotificationID string            `json:"notification_id"`
	ClientID       string            `json:"client_id"`
	AlertID        string            `json:"alert_id"`
	Severity       string            `json:"severity"`
	Source         string            `json:"source"`
	Name           string            `json:"name"`
	Context        map[string]string `json:"context"`
	RuleIDs        []string          `json:"rule_ids"`
	Status         string            `json:"status"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// DB wraps a database connection and provides client, rule, and endpoint operations.
type DB struct {
	conn *sql.DB
}

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

// ============================================================================
// Client Operations
// ============================================================================

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

// ============================================================================
// Rule Operations
// ============================================================================

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

// ============================================================================
// Endpoint Operations
// ============================================================================

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

// ============================================================================
// Notification Operations
// ============================================================================

// GetNotification retrieves a notification by ID.
func (db *DB) GetNotification(ctx context.Context, notificationID string) (*Notification, error) {
	query := `
		SELECT notification_id, client_id, alert_id, severity, source, name, context, rule_ids, status, created_at, updated_at
		FROM notifications
		WHERE notification_id = $1
	`
	var notif Notification
	var contextJSON sql.NullString
	err := db.conn.QueryRowContext(ctx, query, notificationID).Scan(
		&notif.NotificationID,
		&notif.ClientID,
		&notif.AlertID,
		&notif.Severity,
		&notif.Source,
		&notif.Name,
		&contextJSON,
		pq.Array(&notif.RuleIDs),
		&notif.Status,
		&notif.CreatedAt,
		&notif.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("notification not found: %s", notificationID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	notif.Context = unmarshalNotificationContext(contextJSON, "notification_id", notificationID)

	return &notif, nil
}

// ListNotifications retrieves all notifications, optionally filtered by client_id or status.
func (db *DB) ListNotifications(ctx context.Context, clientID *string, status *string) ([]*Notification, error) {
	var query string
	var args []interface{}

	if clientID != nil && status != nil {
		query = `
			SELECT notification_id, client_id, alert_id, severity, source, name, context, rule_ids, status, created_at, updated_at
			FROM notifications
			WHERE client_id = $1 AND status = $2
			ORDER BY created_at DESC
		`
		args = []interface{}{*clientID, *status}
	} else if clientID != nil {
		query = `
			SELECT notification_id, client_id, alert_id, severity, source, name, context, rule_ids, status, created_at, updated_at
			FROM notifications
			WHERE client_id = $1
			ORDER BY created_at DESC
		`
		args = []interface{}{*clientID}
	} else if status != nil {
		query = `
			SELECT notification_id, client_id, alert_id, severity, source, name, context, rule_ids, status, created_at, updated_at
			FROM notifications
			WHERE status = $1
			ORDER BY created_at DESC
		`
		args = []interface{}{*status}
	} else {
		query = `
			SELECT notification_id, client_id, alert_id, severity, source, name, context, rule_ids, status, created_at, updated_at
			FROM notifications
			ORDER BY created_at DESC
		`
		args = []interface{}{}
	}

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list notifications: %w", err)
	}
	defer rows.Close()

	var notifications []*Notification
	for rows.Next() {
		var notif Notification
		var contextJSON sql.NullString
		if err := rows.Scan(
			&notif.NotificationID,
			&notif.ClientID,
			&notif.AlertID,
			&notif.Severity,
			&notif.Source,
			&notif.Name,
			&contextJSON,
			pq.Array(&notif.RuleIDs),
			&notif.Status,
			&notif.CreatedAt,
			&notif.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}

		notif.Context = unmarshalNotificationContext(contextJSON)

		notifications = append(notifications, &notif)
	}
	return notifications, rows.Err()
}
