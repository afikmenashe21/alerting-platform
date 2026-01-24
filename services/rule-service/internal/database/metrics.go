// Package database provides database operations for clients, rules, and endpoints.
package database

import (
	"context"
	"fmt"
	"time"
)

// SystemMetrics holds aggregated metrics from the database.
type SystemMetrics struct {
	// Notification metrics
	TotalNotifications int64            `json:"total_notifications"`
	NotificationsByStatus map[string]int64 `json:"notifications_by_status"`
	NotificationsLast24h int64           `json:"notifications_last_24h"`
	NotificationsLastHour int64          `json:"notifications_last_hour"`

	// Rule metrics
	TotalRules       int64 `json:"total_rules"`
	EnabledRules     int64 `json:"enabled_rules"`
	DisabledRules    int64 `json:"disabled_rules"`

	// Client metrics
	TotalClients     int64 `json:"total_clients"`

	// Endpoint metrics
	TotalEndpoints   int64            `json:"total_endpoints"`
	EndpointsByType  map[string]int64 `json:"endpoints_by_type"`
	EnabledEndpoints int64            `json:"enabled_endpoints"`

	// Time-series data (last 24 hours, hourly buckets)
	NotificationsByHour []HourlyCount `json:"notifications_by_hour"`

	// Timestamp
	CollectedAt time.Time `json:"collected_at"`
}

// HourlyCount represents notification count for a specific hour.
type HourlyCount struct {
	Hour  string `json:"hour"`  // ISO8601 format
	Count int64  `json:"count"`
}

// GetSystemMetrics aggregates metrics from all tables.
func (db *DB) GetSystemMetrics(ctx context.Context) (*SystemMetrics, error) {
	metrics := &SystemMetrics{
		NotificationsByStatus: make(map[string]int64),
		EndpointsByType:       make(map[string]int64),
		NotificationsByHour:   make([]HourlyCount, 0),
		CollectedAt:           time.Now().UTC(),
	}

	// Get notification counts by status
	statusQuery := `
		SELECT status, COUNT(*) as count 
		FROM notifications 
		GROUP BY status
	`
	rows, err := db.conn.QueryContext(ctx, statusQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query notification status: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan status count: %w", err)
		}
		metrics.NotificationsByStatus[status] = count
		metrics.TotalNotifications += count
	}

	// Get notifications in last 24 hours
	last24hQuery := `
		SELECT COUNT(*) FROM notifications 
		WHERE created_at >= NOW() - INTERVAL '24 hours'
	`
	if err := db.conn.QueryRowContext(ctx, last24hQuery).Scan(&metrics.NotificationsLast24h); err != nil {
		return nil, fmt.Errorf("failed to query last 24h notifications: %w", err)
	}

	// Get notifications in last hour
	lastHourQuery := `
		SELECT COUNT(*) FROM notifications 
		WHERE created_at >= NOW() - INTERVAL '1 hour'
	`
	if err := db.conn.QueryRowContext(ctx, lastHourQuery).Scan(&metrics.NotificationsLastHour); err != nil {
		return nil, fmt.Errorf("failed to query last hour notifications: %w", err)
	}

	// Get hourly notification counts for last 24 hours
	hourlyQuery := `
		SELECT 
			date_trunc('hour', created_at) as hour,
			COUNT(*) as count
		FROM notifications 
		WHERE created_at >= NOW() - INTERVAL '24 hours'
		GROUP BY date_trunc('hour', created_at)
		ORDER BY hour ASC
	`
	hourlyRows, err := db.conn.QueryContext(ctx, hourlyQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query hourly notifications: %w", err)
	}
	defer hourlyRows.Close()

	for hourlyRows.Next() {
		var hour time.Time
		var count int64
		if err := hourlyRows.Scan(&hour, &count); err != nil {
			return nil, fmt.Errorf("failed to scan hourly count: %w", err)
		}
		metrics.NotificationsByHour = append(metrics.NotificationsByHour, HourlyCount{
			Hour:  hour.Format(time.RFC3339),
			Count: count,
		})
	}

	// Get rule counts
	rulesQuery := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE enabled = true) as enabled,
			COUNT(*) FILTER (WHERE enabled = false) as disabled
		FROM rules
	`
	if err := db.conn.QueryRowContext(ctx, rulesQuery).Scan(
		&metrics.TotalRules,
		&metrics.EnabledRules,
		&metrics.DisabledRules,
	); err != nil {
		return nil, fmt.Errorf("failed to query rules: %w", err)
	}

	// Get client count
	clientsQuery := `SELECT COUNT(*) FROM clients`
	if err := db.conn.QueryRowContext(ctx, clientsQuery).Scan(&metrics.TotalClients); err != nil {
		return nil, fmt.Errorf("failed to query clients: %w", err)
	}

	// Get endpoint counts by type
	endpointsQuery := `
		SELECT 
			type, 
			COUNT(*) as count,
			COUNT(*) FILTER (WHERE enabled = true) as enabled_count
		FROM endpoints 
		GROUP BY type
	`
	endpointRows, err := db.conn.QueryContext(ctx, endpointsQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query endpoints: %w", err)
	}
	defer endpointRows.Close()

	for endpointRows.Next() {
		var endpointType string
		var count, enabledCount int64
		if err := endpointRows.Scan(&endpointType, &count, &enabledCount); err != nil {
			return nil, fmt.Errorf("failed to scan endpoint count: %w", err)
		}
		metrics.EndpointsByType[endpointType] = count
		metrics.TotalEndpoints += count
		metrics.EnabledEndpoints += enabledCount
	}

	return metrics, nil
}
