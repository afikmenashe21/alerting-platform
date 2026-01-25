// Package database provides database operations for the metrics-service.
package database

import (
	"context"
	"sync"
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
// Reduced to 2 seconds to ensure fast API responses
const queryTimeout = 2 * time.Second

// GetSystemMetrics aggregates metrics from all tables.
// Uses approximate counts from pg_stat for large tables and runs queries in parallel.
func (db *DB) GetSystemMetrics(ctx context.Context) (*SystemMetrics, error) {
	metrics := &SystemMetrics{
		NotificationsByStatus: make(map[string]int64),
		EndpointsByType:       make(map[string]int64),
		NotificationsByHour:   make([]HourlyCount, 0),
		CollectedAt:           time.Now().UTC(),
	}

	// Mutex to protect concurrent writes to metrics
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Helper to create a context with query timeout
	queryCtx := func() (context.Context, context.CancelFunc) {
		return context.WithTimeout(ctx, queryTimeout)
	}

	// Query 1: Approximate row counts (fast, run first as other queries depend on it)
	approxCtx, approxCancel := queryCtx()
	approxQuery := `
		SELECT relname, n_live_tup
		FROM pg_stat_user_tables
		WHERE relname IN ('notifications', 'rules', 'endpoints', 'clients')
	`
	approxRows, err := db.conn.QueryContext(approxCtx, approxQuery)
	if err == nil {
		for approxRows.Next() {
			var tableName string
			var count int64
			if err := approxRows.Scan(&tableName, &count); err == nil {
				switch tableName {
				case "notifications":
					metrics.TotalNotifications = count
				case "rules":
					metrics.TotalRules = count
				case "endpoints":
					metrics.TotalEndpoints = count
				case "clients":
					metrics.TotalClients = count
				}
			}
		}
		approxRows.Close()
	}
	approxCancel()

	// Run remaining queries in parallel
	// Query 2: Status sampling (needs TotalNotifications from query 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		statusCtx, statusCancel := queryCtx()
		defer statusCancel()
		statusQuery := `
			WITH recent AS (
				SELECT status FROM notifications
				ORDER BY created_at DESC
				LIMIT 1000
			)
			SELECT status, COUNT(*) as count FROM recent GROUP BY status
		`
		statusRows, err := db.conn.QueryContext(statusCtx, statusQuery)
		if err == nil {
			defer statusRows.Close()
			var totalSampled int64
			sampleCounts := make(map[string]int64)
			for statusRows.Next() {
				var status string
				var count int64
				if err := statusRows.Scan(&status, &count); err == nil {
					sampleCounts[status] = count
					totalSampled += count
				}
			}
			mu.Lock()
			if totalSampled > 0 && metrics.TotalNotifications > 0 {
				for status, count := range sampleCounts {
					metrics.NotificationsByStatus[status] = (count * metrics.TotalNotifications) / totalSampled
				}
			}
			mu.Unlock()
		}
	}()

	// Query 3: Last 24h count
	wg.Add(1)
	go func() {
		defer wg.Done()
		last24hCtx, last24hCancel := queryCtx()
		defer last24hCancel()
		last24hQuery := `
			SELECT COUNT(*) FROM notifications
			WHERE created_at >= NOW() - INTERVAL '24 hours'
		`
		var count int64
		if err := db.conn.QueryRowContext(last24hCtx, last24hQuery).Scan(&count); err == nil {
			mu.Lock()
			metrics.NotificationsLast24h = count
			mu.Unlock()
		}
	}()

	// Query 4: Last hour count
	wg.Add(1)
	go func() {
		defer wg.Done()
		lastHourCtx, lastHourCancel := queryCtx()
		defer lastHourCancel()
		lastHourQuery := `
			SELECT COUNT(*) FROM notifications
			WHERE created_at >= NOW() - INTERVAL '1 hour'
		`
		var count int64
		if err := db.conn.QueryRowContext(lastHourCtx, lastHourQuery).Scan(&count); err == nil {
			mu.Lock()
			metrics.NotificationsLastHour = count
			mu.Unlock()
		}
	}()

	// Query 5: Hourly breakdown
	wg.Add(1)
	go func() {
		defer wg.Done()
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
			var hourlyData []HourlyCount
			for hourlyRows.Next() {
				var hour time.Time
				var count int64
				if err := hourlyRows.Scan(&hour, &count); err == nil {
					hourlyData = append(hourlyData, HourlyCount{
						Hour:  hour.Format(time.RFC3339),
						Count: count,
					})
				}
			}
			mu.Lock()
			metrics.NotificationsByHour = hourlyData
			mu.Unlock()
		}
	}()

	// Query 6: Rule enabled/disabled counts
	wg.Add(1)
	go func() {
		defer wg.Done()
		rulesCtx, rulesCancel := queryCtx()
		defer rulesCancel()
		rulesQuery := `
			SELECT
				COUNT(*) FILTER (WHERE enabled = true) as enabled,
				COUNT(*) FILTER (WHERE enabled = false) as disabled
			FROM rules
		`
		var enabled, disabled int64
		if err := db.conn.QueryRowContext(rulesCtx, rulesQuery).Scan(&enabled, &disabled); err == nil {
			mu.Lock()
			metrics.EnabledRules = enabled
			metrics.DisabledRules = disabled
			mu.Unlock()
		}
	}()

	// Query 7: Endpoint counts by type
	wg.Add(1)
	go func() {
		defer wg.Done()
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
			endpointsByType := make(map[string]int64)
			var totalEndpoints, enabledEndpoints int64
			for endpointRows.Next() {
				var endpointType string
				var count, enabledCount int64
				if err := endpointRows.Scan(&endpointType, &count, &enabledCount); err == nil {
					endpointsByType[endpointType] = count
					totalEndpoints += count
					enabledEndpoints += enabledCount
				}
			}
			mu.Lock()
			if totalEndpoints > 0 {
				metrics.EndpointsByType = endpointsByType
				metrics.TotalEndpoints = totalEndpoints
				metrics.EnabledEndpoints = enabledEndpoints
			}
			mu.Unlock()
		}
	}()

	// Wait for all parallel queries to complete
	wg.Wait()

	return metrics, nil
}
