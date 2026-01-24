// Package database provides database operations for the metrics-service.
package database

import (
	"context"
	"time"
)

// SystemMetrics holds aggregated metrics from the database.
type SystemMetrics struct {
	// Notification metrics
	TotalNotifications    int64            `json:"total_notifications"`
	NotificationsByStatus map[string]int64 `json:"notifications_by_status"`
	NotificationsLast24h  int64            `json:"notifications_last_24h"`
	NotificationsLastHour int64            `json:"notifications_last_hour"`

	// Rule metrics
	TotalRules    int64 `json:"total_rules"`
	EnabledRules  int64 `json:"enabled_rules"`
	DisabledRules int64 `json:"disabled_rules"`

	// Client metrics
	TotalClients int64 `json:"total_clients"`

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

// queryTimeout is the maximum time for each database query
const queryTimeout = 10 * time.Second

// GetSystemMetrics aggregates metrics from all tables.
// Uses a per-query timeout to prevent long-running queries from blocking.
// Note: Requires idx_notifications_status and idx_notifications_created_at indexes for good performance.
func (db *DB) GetSystemMetrics(ctx context.Context) (*SystemMetrics, error) {
	metrics := &SystemMetrics{
		NotificationsByStatus: make(map[string]int64),
		EndpointsByType:       make(map[string]int64),
		NotificationsByHour:   make([]HourlyCount, 0),
		CollectedAt:           time.Now().UTC(),
	}

	// Helper to create a context with query timeout
	queryCtx := func() (context.Context, context.CancelFunc) {
		return context.WithTimeout(ctx, queryTimeout)
	}

	// Get notification counts by status (uses idx_notifications_status)
	// This query benefits from the status index
	statusCtx, statusCancel := queryCtx()
	defer statusCancel()
	statusQuery := `
		SELECT status, COUNT(*) as count
		FROM notifications
		GROUP BY status
	`
	rows, err := db.conn.QueryContext(statusCtx, statusQuery)
	if err != nil {
		// Don't fail completely - return partial data
		metrics.NotificationsByStatus["error"] = -1
	} else {
		defer rows.Close()
		for rows.Next() {
			var status string
			var count int64
			if err := rows.Scan(&status, &count); err == nil {
				metrics.NotificationsByStatus[status] = count
				metrics.TotalNotifications += count
			}
		}
	}

	// Get notifications in last 24 hours (uses idx_notifications_created_at)
	last24hCtx, last24hCancel := queryCtx()
	defer last24hCancel()
	last24hQuery := `
		SELECT COUNT(*) FROM notifications
		WHERE created_at >= NOW() - INTERVAL '24 hours'
	`
	_ = db.conn.QueryRowContext(last24hCtx, last24hQuery).Scan(&metrics.NotificationsLast24h)

	// Get notifications in last hour (uses idx_notifications_created_at)
	lastHourCtx, lastHourCancel := queryCtx()
	defer lastHourCancel()
	lastHourQuery := `
		SELECT COUNT(*) FROM notifications
		WHERE created_at >= NOW() - INTERVAL '1 hour'
	`
	_ = db.conn.QueryRowContext(lastHourCtx, lastHourQuery).Scan(&metrics.NotificationsLastHour)

	// Get hourly notification counts for last 24 hours (uses idx_notifications_created_at)
	hourlyCtx, hourlyCancel := queryCtx()
	defer hourlyCancel()
	hourlyQuery := `
		SELECT
			date_trunc('hour', created_at) as hour,
			COUNT(*) as count
		FROM notifications
		WHERE created_at >= NOW() - INTERVAL '24 hours'
		GROUP BY date_trunc('hour', created_at)
		ORDER BY hour ASC
	`
	hourlyRows, err := db.conn.QueryContext(hourlyCtx, hourlyQuery)
	if err == nil {
		defer hourlyRows.Close()
		for hourlyRows.Next() {
			var hour time.Time
			var count int64
			if err := hourlyRows.Scan(&hour, &count); err == nil {
				metrics.NotificationsByHour = append(metrics.NotificationsByHour, HourlyCount{
					Hour:  hour.Format(time.RFC3339),
					Count: count,
				})
			}
		}
	}

	// Get rule counts
	rulesCtx, rulesCancel := queryCtx()
	defer rulesCancel()
	rulesQuery := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE enabled = true) as enabled,
			COUNT(*) FILTER (WHERE enabled = false) as disabled
		FROM rules
	`
	_ = db.conn.QueryRowContext(rulesCtx, rulesQuery).Scan(
		&metrics.TotalRules,
		&metrics.EnabledRules,
		&metrics.DisabledRules,
	)

	// Get client count
	clientsCtx, clientsCancel := queryCtx()
	defer clientsCancel()
	clientsQuery := `SELECT COUNT(*) FROM clients`
	_ = db.conn.QueryRowContext(clientsCtx, clientsQuery).Scan(&metrics.TotalClients)

	// Get endpoint counts by type
	endpointsCtx, endpointsCancel := queryCtx()
	defer endpointsCancel()
	endpointsQuery := `
		SELECT
			type,
			COUNT(*) as count,
			COUNT(*) FILTER (WHERE enabled = true) as enabled_count
		FROM endpoints
		GROUP BY type
	`
	endpointRows, err := db.conn.QueryContext(endpointsCtx, endpointsQuery)
	if err == nil {
		defer endpointRows.Close()
		for endpointRows.Next() {
			var endpointType string
			var count, enabledCount int64
			if err := endpointRows.Scan(&endpointType, &count, &enabledCount); err == nil {
				metrics.EndpointsByType[endpointType] = count
				metrics.TotalEndpoints += count
				metrics.EnabledEndpoints += enabledCount
			}
		}
	}

	return metrics, nil
}
