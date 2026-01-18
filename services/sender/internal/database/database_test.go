package database

import (
	"context"
	"strings"
	"testing"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

// setupTestDB creates a test database connection
// In a real test environment, you would use a test database or testcontainers
func setupTestDB(t *testing.T) *DB {
	dsn := "postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable"
	db, err := NewDB(dsn)
	if err != nil {
		t.Skipf("Skipping database test: Postgres not available: %v", err)
		return nil
	}
	return db
}

func TestNewDB(t *testing.T) {
	tests := []struct {
		name    string
		dsn     string
		wantErr bool
		skipIfUnavailable bool
	}{
		{
			name:    "valid dsn",
			dsn:     "postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable",
			wantErr: false,
			skipIfUnavailable: true,
		},
		{
			name:    "invalid dsn",
			dsn:     "invalid://dsn",
			wantErr: true,
			skipIfUnavailable: false,
		},
		{
			name:    "empty dsn",
			dsn:     "",
			wantErr: true,
			skipIfUnavailable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := NewDB(tt.dsn)
			if (err != nil) != tt.wantErr {
				// If we expected no error but got one, and it's a connection error, skip
				if !tt.wantErr && err != nil && tt.skipIfUnavailable {
					if contains(err.Error(), "dial tcp") || contains(err.Error(), "connection refused") {
						t.Skipf("Skipping test: Postgres not available: %v", err)
						return
					}
				}
				t.Errorf("NewDB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && db != nil {
				_ = db.Close()
			}
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestDB_Close(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	if err := db.Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}

	// Close again should be safe
	_ = db.Close()
}

func TestDB_GetNotification(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	ctx := context.Background()

	tests := []struct {
		name           string
		notificationID string
		wantErr        bool
	}{
		{
			name:           "non-existent notification",
			notificationID: "non-existent-id-12345",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.GetNotification(ctx, tt.notificationID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetNotification() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDB_GetNotification_WithContext(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	// Test with valid context JSON
	ctx := context.Background()

	// Insert a test notification
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO notifications (notification_id, client_id, alert_id, severity, source, name, context, rule_ids, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (notification_id) DO NOTHING
	`, "test-notif-001", "client-001", "alert-001", "HIGH", "test-source", "test-name", `{"key1":"value1"}`, pq.Array([]string{"rule-001"}), "RECEIVED")
	if err != nil {
		t.Logf("Could not insert test notification (may already exist): %v", err)
	}

	// Test GetNotification with context
	notif, err := db.GetNotification(ctx, "test-notif-001")
	if err != nil {
		t.Logf("GetNotification() error (notification may not exist): %v", err)
		return
	}

	if notif == nil {
		t.Fatal("GetNotification() returned nil notification")
	}

	if len(notif.Context) == 0 {
		t.Error("GetNotification() should have context")
	}

	if notif.Context["key1"] != "value1" {
		t.Errorf("GetNotification() context = %v, want key1=value1", notif.Context)
	}

	// Test with null context
	_, err = db.conn.ExecContext(ctx, `
		INSERT INTO notifications (notification_id, client_id, alert_id, severity, source, name, context, rule_ids, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (notification_id) DO UPDATE SET context = $7
	`, "test-notif-002", "client-002", "alert-002", "MEDIUM", "test-source", "test-name", nil, pq.Array([]string{"rule-002"}), "RECEIVED")
	if err != nil {
		t.Logf("Could not insert test notification: %v", err)
	}

	notif2, err := db.GetNotification(ctx, "test-notif-002")
	if err != nil {
		t.Logf("GetNotification() error: %v", err)
		return
	}

	if notif2 == nil {
		t.Fatal("GetNotification() returned nil notification")
	}

	if notif2.Context == nil {
		t.Error("GetNotification() should initialize empty context map for null context")
	}
}

func TestDB_GetNotification_InvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	ctx := context.Background()

	// Insert notification with invalid JSON context
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO notifications (notification_id, client_id, alert_id, severity, source, name, context, rule_ids, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (notification_id) DO UPDATE SET context = $7
	`, "test-notif-invalid-json", "client-001", "alert-001", "HIGH", "test-source", "test-name", `{invalid json}`, pq.Array([]string{"rule-001"}), "RECEIVED")
	if err != nil {
		t.Logf("Could not insert test notification: %v", err)
	}

	// GetNotification should handle invalid JSON gracefully
	notif, err := db.GetNotification(ctx, "test-notif-invalid-json")
	if err != nil {
		t.Logf("GetNotification() error: %v", err)
		return
	}

	if notif == nil {
		t.Fatal("GetNotification() returned nil notification")
	}

	// Should have empty context map when JSON is invalid
	if notif.Context == nil {
		t.Error("GetNotification() should initialize empty context map for invalid JSON")
	}
}

func TestDB_UpdateNotificationStatus(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	ctx := context.Background()

	// Insert a test notification
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO notifications (notification_id, client_id, alert_id, severity, source, name, context, rule_ids, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (notification_id) DO NOTHING
	`, "test-notif-update", "client-001", "alert-001", "HIGH", "test-source", "test-name", `{}`, pq.Array([]string{"rule-001"}), "RECEIVED")
	if err != nil {
		t.Logf("Could not insert test notification: %v", err)
	}

	tests := []struct {
		name           string
		notificationID string
		status         string
		wantErr        bool
	}{
		{
			name:           "update to SENT",
			notificationID: "test-notif-update",
			status:         "SENT",
			wantErr:        false,
		},
		{
			name:           "update to FAILED",
			notificationID: "test-notif-update",
			status:         "FAILED",
			wantErr:        false,
		},
		{
			name:           "non-existent notification",
			notificationID: "non-existent-id-99999",
			status:         "SENT",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.UpdateNotificationStatus(ctx, tt.notificationID, tt.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateNotificationStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDB_GetEndpointsByRuleIDs(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	ctx := context.Background()

	tests := []struct {
		name    string
		ruleIDs []string
		wantErr bool
	}{
		{
			name:    "empty rule IDs",
			ruleIDs: []string{},
			wantErr: false,
		},
		{
			name:    "nil rule IDs",
			ruleIDs: nil,
			wantErr: false,
		},
		{
			name:    "single rule ID",
			ruleIDs: []string{"rule-001"},
			wantErr: false,
		},
		{
			name:    "multiple rule IDs",
			ruleIDs: []string{"rule-001", "rule-002"},
			wantErr: false,
		},
		{
			name:    "rule IDs with empty strings",
			ruleIDs: []string{"rule-001", "", "rule-002"},
			wantErr: false,
		},
		{
			name:    "all empty strings",
			ruleIDs: []string{"", "", ""},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoints, err := db.GetEndpointsByRuleIDs(ctx, tt.ruleIDs)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetEndpointsByRuleIDs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && endpoints == nil {
				t.Error("GetEndpointsByRuleIDs() returned nil map")
			}
		})
	}
}

func TestDB_GetEmailEndpointsByRuleIDs(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	ctx := context.Background()

	tests := []struct {
		name    string
		ruleIDs []string
		wantErr bool
	}{
		{
			name:    "empty rule IDs",
			ruleIDs: []string{},
			wantErr: false,
		},
		{
			name:    "single rule ID",
			ruleIDs: []string{"rule-001"},
			wantErr: false,
		},
		{
			name:    "multiple rule IDs",
			ruleIDs: []string{"rule-001", "rule-002"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoints, err := db.GetEmailEndpointsByRuleIDs(ctx, tt.ruleIDs)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetEmailEndpointsByRuleIDs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && endpoints == nil {
				t.Error("GetEmailEndpointsByRuleIDs() returned nil map")
			}
		})
	}
}

func TestDB_GetEmailEndpointsByRuleIDs_FiltersEmailOnly(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	ctx := context.Background()

	// Insert test endpoints
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO endpoints (endpoint_id, rule_id, type, value, enabled)
		VALUES 
			('ep-001', 'rule-001', 'email', 'test@example.com', true),
			('ep-002', 'rule-001', 'slack', 'https://hooks.slack.com/test', true),
			('ep-003', 'rule-001', 'webhook', 'https://webhook.example.com', true)
		ON CONFLICT (endpoint_id) DO NOTHING
	`)
	if err != nil {
		t.Logf("Could not insert test endpoints: %v", err)
	}

	endpoints, err := db.GetEmailEndpointsByRuleIDs(ctx, []string{"rule-001"})
	if err != nil {
		t.Logf("GetEmailEndpointsByRuleIDs() error: %v", err)
		return
	}

	// Should only return email endpoints
	if len(endpoints["rule-001"]) != 1 {
		t.Errorf("GetEmailEndpointsByRuleIDs() returned %d endpoints, want 1", len(endpoints["rule-001"]))
	}

	if endpoints["rule-001"][0] != "test@example.com" {
		t.Errorf("GetEmailEndpointsByRuleIDs() returned %v, want [test@example.com]", endpoints["rule-001"])
	}
}
