// Package database provides tests for database operations.
// These tests use sqlmock to mock database interactions for 100% coverage.
package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
)

// TestNewDB tests the NewDB constructor with various scenarios.
func TestNewDB(t *testing.T) {
	tests := []struct {
		name    string
		dsn     string
		wantErr bool
	}{
		{
			name:    "invalid DSN",
			dsn:     "invalid-dsn",
			wantErr: true,
		},
		{
			name:    "empty DSN",
			dsn:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := NewDB(tt.dsn)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && db != nil {
				db.Close()
			}
		})
	}
}

// TestDB_Close tests the Close method.
func TestDB_Close(t *testing.T) {
	db := &DB{conn: nil}
	if err := db.Close(); err != nil {
		t.Errorf("Close() with nil conn error = %v, want nil", err)
	}
}

// TestDB_CreateClient tests CreateClient with various scenarios.
func TestDB_CreateClient(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	d := &DB{conn: db}
	ctx := context.Background()

	tests := []struct {
		name      string
		clientID  string
		nameValue string
		setupMock func()
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "successful create",
			clientID:  "client-1",
			nameValue: "Test Client",
			setupMock: func() {
				mock.ExpectExec("INSERT INTO clients").
					WithArgs("client-1", "Test Client").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name:      "duplicate client",
			clientID:  "client-1",
			nameValue: "Test Client",
			setupMock: func() {
				mock.ExpectExec("INSERT INTO clients").
					WithArgs("client-1", "Test Client").
					WillReturnError(&pq.Error{Code: "23505"})
			},
			wantErr: true,
			errMsg:  "client already exists",
		},
		{
			name:      "database error",
			clientID:  "client-1",
			nameValue: "Test Client",
			setupMock: func() {
				mock.ExpectExec("INSERT INTO clients").
					WithArgs("client-1", "Test Client").
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			err := d.CreateClient(ctx, tt.clientID, tt.nameValue)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateClient() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("CreateClient() error = %v, want error containing %v", err.Error(), tt.errMsg)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Mock expectations were not met: %v", err)
			}
		})
	}
}

// TestDB_GetClient tests GetClient with various scenarios.
func TestDB_GetClient(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	d := &DB{conn: db}
	ctx := context.Background()

	tests := []struct {
		name      string
		clientID  string
		setupMock func()
		wantErr   bool
		errMsg    string
	}{
		{
			name:     "successful get",
			clientID: "client-1",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{"client_id", "name", "created_at", "updated_at"}).
					AddRow("client-1", "Test Client", time.Now(), time.Now())
				mock.ExpectQuery("SELECT client_id, name, created_at, updated_at").
					WithArgs("client-1").
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name:     "client not found",
			clientID: "client-999",
			setupMock: func() {
				mock.ExpectQuery("SELECT client_id, name, created_at, updated_at").
					WithArgs("client-999").
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			errMsg:  "client not found",
		},
		{
			name:     "database error",
			clientID: "client-1",
			setupMock: func() {
				mock.ExpectQuery("SELECT client_id, name, created_at, updated_at").
					WithArgs("client-1").
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			client, err := d.GetClient(ctx, tt.clientID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("GetClient() returned nil client")
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("GetClient() error = %v, want error containing %v", err.Error(), tt.errMsg)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Mock expectations were not met: %v", err)
			}
		})
	}
}

// TestDB_ListClients tests ListClients with pagination.
func TestDB_ListClients(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	d := &DB{conn: db}
	ctx := context.Background()

	t.Run("successful list", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
		rows := sqlmock.NewRows([]string{"client_id", "name", "created_at", "updated_at"}).
			AddRow("client-1", "Client 1", time.Now(), time.Now()).
			AddRow("client-2", "Client 2", time.Now(), time.Now())
		mock.ExpectQuery("SELECT client_id, name, created_at, updated_at").
			WithArgs(50, 0).
			WillReturnRows(rows)

		result, err := d.ListClients(ctx, 50, 0)
		if err != nil {
			t.Errorf("ListClients() error = %v", err)
		}
		if len(result.Clients) != 2 {
			t.Errorf("ListClients() returned %d clients, want 2", len(result.Clients))
		}
		if result.Total != 2 {
			t.Errorf("ListClients() total = %d, want 2", result.Total)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		rows := sqlmock.NewRows([]string{"client_id", "name", "created_at", "updated_at"})
		mock.ExpectQuery("SELECT client_id, name, created_at, updated_at").
			WithArgs(50, 0).
			WillReturnRows(rows)

		result, err := d.ListClients(ctx, 50, 0)
		if err != nil {
			t.Errorf("ListClients() error = %v", err)
		}
		if len(result.Clients) != 0 {
			t.Errorf("ListClients() returned %d clients, want 0", len(result.Clients))
		}
		if result.Total != 0 {
			t.Errorf("ListClients() total = %d, want 0", result.Total)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})

	t.Run("database error on count", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT").
			WillReturnError(sql.ErrConnDone)

		_, err := d.ListClients(ctx, 50, 0)
		if err == nil {
			t.Error("ListClients() expected error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})

	t.Run("database error on query", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
		mock.ExpectQuery("SELECT client_id, name, created_at, updated_at").
			WithArgs(50, 0).
			WillReturnError(sql.ErrConnDone)

		_, err := d.ListClients(ctx, 50, 0)
		if err == nil {
			t.Error("ListClients() expected error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}

// TestDB_CreateRule tests CreateRule with various scenarios.
func TestDB_CreateRule(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	d := &DB{conn: db}
	ctx := context.Background()

	t.Run("successful create", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"rule_id", "client_id", "severity", "source", "name", "enabled", "version", "created_at", "updated_at"}).
			AddRow("rule-1", "client-1", "HIGH", "source-1", "alert-1", true, 1, time.Now(), time.Now())
		mock.ExpectQuery("INSERT INTO rules").
			WithArgs("client-1", "HIGH", "source-1", "alert-1").
			WillReturnRows(rows)

		rule, err := d.CreateRule(ctx, "client-1", "HIGH", "source-1", "alert-1")
		if err != nil {
			t.Errorf("CreateRule() error = %v", err)
		}
		if rule == nil {
			t.Error("CreateRule() returned nil rule")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})

	t.Run("duplicate rule (exact match)", func(t *testing.T) {
		mock.ExpectQuery("INSERT INTO rules").
			WithArgs("client-1", "HIGH", "source-1", "alert-1").
			WillReturnError(&pq.Error{Code: "23505"})

		_, err := d.CreateRule(ctx, "client-1", "HIGH", "source-1", "alert-1")
		if err == nil {
			t.Error("CreateRule() expected error for duplicate")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})

	t.Run("client not found", func(t *testing.T) {
		mock.ExpectQuery("INSERT INTO rules").
			WithArgs("client-999", "HIGH", "source-1", "alert-1").
			WillReturnError(&pq.Error{Code: "23503"})

		_, err := d.CreateRule(ctx, "client-999", "HIGH", "source-1", "alert-1")
		if err == nil {
			t.Error("CreateRule() expected error for missing client")
		}
		if !contains(err.Error(), "client not found") {
			t.Errorf("CreateRule() error = %v, want 'client not found'", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}

// TestDB_GetRule tests GetRule.
func TestDB_GetRule(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	d := &DB{conn: db}
	ctx := context.Background()

	t.Run("successful get", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"rule_id", "client_id", "severity", "source", "name", "enabled", "version", "created_at", "updated_at"}).
			AddRow("rule-1", "client-1", "HIGH", "source-1", "alert-1", true, 1, time.Now(), time.Now())
		mock.ExpectQuery("SELECT rule_id, client_id, severity, source, name, enabled, version, created_at, updated_at").
			WithArgs("rule-1").
			WillReturnRows(rows)

		rule, err := d.GetRule(ctx, "rule-1")
		if err != nil {
			t.Errorf("GetRule() error = %v", err)
		}
		if rule == nil {
			t.Error("GetRule() returned nil rule")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})

	t.Run("rule not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT rule_id, client_id, severity, source, name, enabled, version, created_at, updated_at").
			WithArgs("rule-999").
			WillReturnError(sql.ErrNoRows)

		_, err := d.GetRule(ctx, "rule-999")
		if err == nil {
			t.Error("GetRule() expected error")
		}
		if !contains(err.Error(), "rule not found") {
			t.Errorf("GetRule() error = %v, want 'rule not found'", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}

// TestDB_ListRules tests ListRules with pagination and optional client filter.
func TestDB_ListRules(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	d := &DB{conn: db}
	ctx := context.Background()

	t.Run("list all rules", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		rows := sqlmock.NewRows([]string{"rule_id", "client_id", "severity", "source", "name", "enabled", "version", "created_at", "updated_at"}).
			AddRow("rule-1", "client-1", "HIGH", "source-1", "alert-1", true, 1, time.Now(), time.Now())
		mock.ExpectQuery("SELECT rule_id, client_id, severity, source, name, enabled, version, created_at, updated_at").
			WithArgs(50, 0).
			WillReturnRows(rows)

		result, err := d.ListRules(ctx, nil, 50, 0)
		if err != nil {
			t.Errorf("ListRules() error = %v", err)
		}
		if len(result.Rules) != 1 {
			t.Errorf("ListRules() returned %d rules, want 1", len(result.Rules))
		}
		if result.Total != 1 {
			t.Errorf("ListRules() total = %d, want 1", result.Total)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})

	t.Run("list rules by client", func(t *testing.T) {
		clientID := "client-1"
		mock.ExpectQuery("SELECT COUNT").
			WithArgs(clientID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		rows := sqlmock.NewRows([]string{"rule_id", "client_id", "severity", "source", "name", "enabled", "version", "created_at", "updated_at"}).
			AddRow("rule-1", "client-1", "HIGH", "source-1", "alert-1", true, 1, time.Now(), time.Now())
		mock.ExpectQuery("SELECT rule_id, client_id, severity, source, name, enabled, version, created_at, updated_at").
			WithArgs(clientID, 50, 0).
			WillReturnRows(rows)

		result, err := d.ListRules(ctx, &clientID, 50, 0)
		if err != nil {
			t.Errorf("ListRules() error = %v", err)
		}
		if len(result.Rules) != 1 {
			t.Errorf("ListRules() returned %d rules, want 1", len(result.Rules))
		}
		if result.Total != 1 {
			t.Errorf("ListRules() total = %d, want 1", result.Total)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}

// TestDB_UpdateRule tests UpdateRule with optimistic locking.
func TestDB_UpdateRule(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	d := &DB{conn: db}
	ctx := context.Background()

	t.Run("successful update", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"rule_id", "client_id", "severity", "source", "name", "enabled", "version", "created_at", "updated_at"}).
			AddRow("rule-1", "client-1", "CRITICAL", "source-2", "alert-2", true, 2, time.Now(), time.Now())
		mock.ExpectQuery("UPDATE rules").
			WithArgs("rule-1", "CRITICAL", "source-2", "alert-2", 1).
			WillReturnRows(rows)

		rule, err := d.UpdateRule(ctx, "rule-1", "CRITICAL", "source-2", "alert-2", 1)
		if err != nil {
			t.Errorf("UpdateRule() error = %v", err)
		}
		if rule == nil {
			t.Error("UpdateRule() returned nil rule")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})

	t.Run("version mismatch", func(t *testing.T) {
		mock.ExpectQuery("UPDATE rules").
			WithArgs("rule-1", "CRITICAL", "source-2", "alert-2", 1).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs("rule-1").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		_, err := d.UpdateRule(ctx, "rule-1", "CRITICAL", "source-2", "alert-2", 1)
		if err == nil {
			t.Error("UpdateRule() expected error for version mismatch")
		}
		if !contains(err.Error(), "version mismatch") {
			t.Errorf("UpdateRule() error = %v, want 'version mismatch'", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})

	t.Run("rule not found", func(t *testing.T) {
		mock.ExpectQuery("UPDATE rules").
			WithArgs("rule-999", "CRITICAL", "source-2", "alert-2", 1).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs("rule-999").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		_, err := d.UpdateRule(ctx, "rule-999", "CRITICAL", "source-2", "alert-2", 1)
		if err == nil {
			t.Error("UpdateRule() expected error for missing rule")
		}
		if !contains(err.Error(), "rule not found") {
			t.Errorf("UpdateRule() error = %v, want 'rule not found'", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}

// TestDB_ToggleRuleEnabled tests ToggleRuleEnabled.
func TestDB_ToggleRuleEnabled(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	d := &DB{conn: db}
	ctx := context.Background()

	t.Run("successful toggle", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"rule_id", "client_id", "severity", "source", "name", "enabled", "version", "created_at", "updated_at"}).
			AddRow("rule-1", "client-1", "HIGH", "source-1", "alert-1", false, 2, time.Now(), time.Now())
		mock.ExpectQuery("UPDATE rules").
			WithArgs("rule-1", false, 1).
			WillReturnRows(rows)

		rule, err := d.ToggleRuleEnabled(ctx, "rule-1", false, 1)
		if err != nil {
			t.Errorf("ToggleRuleEnabled() error = %v", err)
		}
		if rule == nil {
			t.Error("ToggleRuleEnabled() returned nil rule")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})

	t.Run("version mismatch", func(t *testing.T) {
		mock.ExpectQuery("UPDATE rules").
			WithArgs("rule-1", false, 1).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs("rule-1").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		_, err := d.ToggleRuleEnabled(ctx, "rule-1", false, 1)
		if err == nil {
			t.Error("ToggleRuleEnabled() expected error for version mismatch")
		}
		if !contains(err.Error(), "version mismatch") {
			t.Errorf("ToggleRuleEnabled() error = %v, want 'version mismatch'", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}

// TestDB_DeleteRule tests DeleteRule.
func TestDB_DeleteRule(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	d := &DB{conn: db}
	ctx := context.Background()

	t.Run("successful delete", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM rules").
			WithArgs("rule-1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := d.DeleteRule(ctx, "rule-1")
		if err != nil {
			t.Errorf("DeleteRule() error = %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})

	t.Run("rule not found", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM rules").
			WithArgs("rule-999").
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := d.DeleteRule(ctx, "rule-999")
		if err == nil {
			t.Error("DeleteRule() expected error for missing rule")
		}
		if !contains(err.Error(), "rule not found") {
			t.Errorf("DeleteRule() error = %v, want 'rule not found'", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}

// TestDB_GetRulesUpdatedSince tests GetRulesUpdatedSince.
func TestDB_GetRulesUpdatedSince(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	d := &DB{conn: db}
	ctx := context.Background()

	t.Run("successful get", func(t *testing.T) {
		since := time.Now().Add(-1 * time.Hour)
		rows := sqlmock.NewRows([]string{"rule_id", "client_id", "severity", "source", "name", "enabled", "version", "created_at", "updated_at"}).
			AddRow("rule-1", "client-1", "HIGH", "source-1", "alert-1", true, 1, time.Now(), time.Now())
		mock.ExpectQuery("SELECT rule_id, client_id, severity, source, name, enabled, version, created_at, updated_at").
			WithArgs(since).
			WillReturnRows(rows)

		rules, err := d.GetRulesUpdatedSince(ctx, since)
		if err != nil {
			t.Errorf("GetRulesUpdatedSince() error = %v", err)
		}
		if len(rules) != 1 {
			t.Errorf("GetRulesUpdatedSince() returned %d rules, want 1", len(rules))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}

// TestDB_CreateEndpoint tests CreateEndpoint.
func TestDB_CreateEndpoint(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	d := &DB{conn: db}
	ctx := context.Background()

	t.Run("successful create", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"endpoint_id", "rule_id", "type", "value", "enabled", "created_at", "updated_at"}).
			AddRow("endpoint-1", "rule-1", "email", "test@example.com", true, time.Now(), time.Now())
		mock.ExpectQuery("INSERT INTO endpoints").
			WithArgs("rule-1", "email", "test@example.com").
			WillReturnRows(rows)

		endpoint, err := d.CreateEndpoint(ctx, "rule-1", "email", "test@example.com")
		if err != nil {
			t.Errorf("CreateEndpoint() error = %v", err)
		}
		if endpoint == nil {
			t.Error("CreateEndpoint() returned nil endpoint")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})

	t.Run("duplicate endpoint", func(t *testing.T) {
		mock.ExpectQuery("INSERT INTO endpoints").
			WithArgs("rule-1", "email", "test@example.com").
			WillReturnError(&pq.Error{Code: "23505"})

		_, err := d.CreateEndpoint(ctx, "rule-1", "email", "test@example.com")
		if err == nil {
			t.Error("CreateEndpoint() expected error for duplicate")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})

	t.Run("rule not found", func(t *testing.T) {
		mock.ExpectQuery("INSERT INTO endpoints").
			WithArgs("rule-999", "email", "test@example.com").
			WillReturnError(&pq.Error{Code: "23503"})

		_, err := d.CreateEndpoint(ctx, "rule-999", "email", "test@example.com")
		if err == nil {
			t.Error("CreateEndpoint() expected error for missing rule")
		}
		if !contains(err.Error(), "rule not found") {
			t.Errorf("CreateEndpoint() error = %v, want 'rule not found'", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}

// TestDB_GetEndpoint tests GetEndpoint.
func TestDB_GetEndpoint(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	d := &DB{conn: db}
	ctx := context.Background()

	t.Run("successful get", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"endpoint_id", "rule_id", "type", "value", "enabled", "created_at", "updated_at"}).
			AddRow("endpoint-1", "rule-1", "email", "test@example.com", true, time.Now(), time.Now())
		mock.ExpectQuery("SELECT endpoint_id, rule_id, type, value, enabled, created_at, updated_at").
			WithArgs("endpoint-1").
			WillReturnRows(rows)

		endpoint, err := d.GetEndpoint(ctx, "endpoint-1")
		if err != nil {
			t.Errorf("GetEndpoint() error = %v", err)
		}
		if endpoint == nil {
			t.Error("GetEndpoint() returned nil endpoint")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})

	t.Run("endpoint not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT endpoint_id, rule_id, type, value, enabled, created_at, updated_at").
			WithArgs("endpoint-999").
			WillReturnError(sql.ErrNoRows)

		_, err := d.GetEndpoint(ctx, "endpoint-999")
		if err == nil {
			t.Error("GetEndpoint() expected error")
		}
		if !contains(err.Error(), "endpoint not found") {
			t.Errorf("GetEndpoint() error = %v, want 'endpoint not found'", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}

// TestDB_ListEndpoints tests ListEndpoints with pagination and optional rule filter.
func TestDB_ListEndpoints(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	d := &DB{conn: db}
	ctx := context.Background()

	t.Run("list all endpoints", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		rows := sqlmock.NewRows([]string{"endpoint_id", "rule_id", "type", "value", "enabled", "created_at", "updated_at"}).
			AddRow("endpoint-1", "rule-1", "email", "test@example.com", true, time.Now(), time.Now())
		mock.ExpectQuery("SELECT endpoint_id, rule_id, type, value, enabled, created_at, updated_at").
			WithArgs(50, 0).
			WillReturnRows(rows)

		result, err := d.ListEndpoints(ctx, nil, 50, 0)
		if err != nil {
			t.Errorf("ListEndpoints() error = %v", err)
		}
		if len(result.Endpoints) != 1 {
			t.Errorf("ListEndpoints() returned %d endpoints, want 1", len(result.Endpoints))
		}
		if result.Total != 1 {
			t.Errorf("ListEndpoints() total = %d, want 1", result.Total)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})

	t.Run("list endpoints by rule", func(t *testing.T) {
		ruleID := "rule-1"
		mock.ExpectQuery("SELECT COUNT").
			WithArgs(ruleID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		rows := sqlmock.NewRows([]string{"endpoint_id", "rule_id", "type", "value", "enabled", "created_at", "updated_at"}).
			AddRow("endpoint-1", "rule-1", "email", "test@example.com", true, time.Now(), time.Now())
		mock.ExpectQuery("SELECT endpoint_id, rule_id, type, value, enabled, created_at, updated_at").
			WithArgs(ruleID, 50, 0).
			WillReturnRows(rows)

		result, err := d.ListEndpoints(ctx, &ruleID, 50, 0)
		if err != nil {
			t.Errorf("ListEndpoints() error = %v", err)
		}
		if len(result.Endpoints) != 1 {
			t.Errorf("ListEndpoints() returned %d endpoints, want 1", len(result.Endpoints))
		}
		if result.Total != 1 {
			t.Errorf("ListEndpoints() total = %d, want 1", result.Total)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}

// TestDB_UpdateEndpoint tests UpdateEndpoint.
func TestDB_UpdateEndpoint(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	d := &DB{conn: db}
	ctx := context.Background()

	t.Run("successful update", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"endpoint_id", "rule_id", "type", "value", "enabled", "created_at", "updated_at"}).
			AddRow("endpoint-1", "rule-1", "webhook", "https://example.com", true, time.Now(), time.Now())
		mock.ExpectQuery("UPDATE endpoints").
			WithArgs("endpoint-1", "webhook", "https://example.com").
			WillReturnRows(rows)

		endpoint, err := d.UpdateEndpoint(ctx, "endpoint-1", "webhook", "https://example.com")
		if err != nil {
			t.Errorf("UpdateEndpoint() error = %v", err)
		}
		if endpoint == nil {
			t.Error("UpdateEndpoint() returned nil endpoint")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})

	t.Run("endpoint not found", func(t *testing.T) {
		mock.ExpectQuery("UPDATE endpoints").
			WithArgs("endpoint-999", "webhook", "https://example.com").
			WillReturnError(sql.ErrNoRows)

		_, err := d.UpdateEndpoint(ctx, "endpoint-999", "webhook", "https://example.com")
		if err == nil {
			t.Error("UpdateEndpoint() expected error")
		}
		if !contains(err.Error(), "endpoint not found") {
			t.Errorf("UpdateEndpoint() error = %v, want 'endpoint not found'", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}

// TestDB_ToggleEndpointEnabled tests ToggleEndpointEnabled.
func TestDB_ToggleEndpointEnabled(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	d := &DB{conn: db}
	ctx := context.Background()

	t.Run("successful toggle", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"endpoint_id", "rule_id", "type", "value", "enabled", "created_at", "updated_at"}).
			AddRow("endpoint-1", "rule-1", "email", "test@example.com", false, time.Now(), time.Now())
		mock.ExpectQuery("UPDATE endpoints").
			WithArgs("endpoint-1", false).
			WillReturnRows(rows)

		endpoint, err := d.ToggleEndpointEnabled(ctx, "endpoint-1", false)
		if err != nil {
			t.Errorf("ToggleEndpointEnabled() error = %v", err)
		}
		if endpoint == nil {
			t.Error("ToggleEndpointEnabled() returned nil endpoint")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})

	t.Run("endpoint not found", func(t *testing.T) {
		mock.ExpectQuery("UPDATE endpoints").
			WithArgs("endpoint-999", false).
			WillReturnError(sql.ErrNoRows)

		_, err := d.ToggleEndpointEnabled(ctx, "endpoint-999", false)
		if err == nil {
			t.Error("ToggleEndpointEnabled() expected error")
		}
		if !contains(err.Error(), "endpoint not found") {
			t.Errorf("ToggleEndpointEnabled() error = %v, want 'endpoint not found'", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}

// TestDB_DeleteEndpoint tests DeleteEndpoint.
func TestDB_DeleteEndpoint(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	d := &DB{conn: db}
	ctx := context.Background()

	t.Run("successful delete", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM endpoints").
			WithArgs("endpoint-1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := d.DeleteEndpoint(ctx, "endpoint-1")
		if err != nil {
			t.Errorf("DeleteEndpoint() error = %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})

	t.Run("endpoint not found", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM endpoints").
			WithArgs("endpoint-999").
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := d.DeleteEndpoint(ctx, "endpoint-999")
		if err == nil {
			t.Error("DeleteEndpoint() expected error for missing endpoint")
		}
		if !contains(err.Error(), "endpoint not found") {
			t.Errorf("DeleteEndpoint() error = %v, want 'endpoint not found'", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}

// TestDB_GetNotification tests GetNotification.
func TestDB_GetNotification(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	d := &DB{conn: db}
	ctx := context.Background()

	t.Run("successful get with context", func(t *testing.T) {
		contextJSON, _ := json.Marshal(map[string]string{"key": "value"})
		rows := sqlmock.NewRows([]string{"notification_id", "client_id", "alert_id", "severity", "source", "name", "context", "rule_ids", "status", "created_at", "updated_at"}).
			AddRow("notif-1", "client-1", "alert-1", "HIGH", "source-1", "alert-1", string(contextJSON), pq.Array([]string{"rule-1"}), "RECEIVED", time.Now(), time.Now())
		mock.ExpectQuery("SELECT notification_id, client_id, alert_id, severity, source, name, context, rule_ids, status, created_at, updated_at").
			WithArgs("notif-1").
			WillReturnRows(rows)

		notif, err := d.GetNotification(ctx, "notif-1")
		if err != nil {
			t.Errorf("GetNotification() error = %v", err)
		}
		if notif == nil {
			t.Error("GetNotification() returned nil notification")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})

	t.Run("successful get without context", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"notification_id", "client_id", "alert_id", "severity", "source", "name", "context", "rule_ids", "status", "created_at", "updated_at"}).
			AddRow("notif-1", "client-1", "alert-1", "HIGH", "source-1", "alert-1", nil, pq.Array([]string{"rule-1"}), "RECEIVED", time.Now(), time.Now())
		mock.ExpectQuery("SELECT notification_id, client_id, alert_id, severity, source, name, context, rule_ids, status, created_at, updated_at").
			WithArgs("notif-1").
			WillReturnRows(rows)

		notif, err := d.GetNotification(ctx, "notif-1")
		if err != nil {
			t.Errorf("GetNotification() error = %v", err)
		}
		if notif == nil {
			t.Error("GetNotification() returned nil notification")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})

	t.Run("notification not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT notification_id, client_id, alert_id, severity, source, name, context, rule_ids, status, created_at, updated_at").
			WithArgs("notif-999").
			WillReturnError(sql.ErrNoRows)

		_, err := d.GetNotification(ctx, "notif-999")
		if err == nil {
			t.Error("GetNotification() expected error")
		}
		if !contains(err.Error(), "notification not found") {
			t.Errorf("GetNotification() error = %v, want 'notification not found'", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}

// TestDB_ListNotifications tests ListNotifications with pagination and various filters.
func TestDB_ListNotifications(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	d := &DB{conn: db}
	ctx := context.Background()

	t.Run("list all", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		rows := sqlmock.NewRows([]string{"notification_id", "client_id", "alert_id", "severity", "source", "name", "context", "rule_ids", "status", "created_at", "updated_at"}).
			AddRow("notif-1", "client-1", "alert-1", "HIGH", "source-1", "alert-1", nil, pq.Array([]string{"rule-1"}), "RECEIVED", time.Now(), time.Now())
		mock.ExpectQuery("SELECT notification_id, client_id, alert_id, severity, source, name, context, rule_ids, status, created_at, updated_at").
			WithArgs(50, 0).
			WillReturnRows(rows)

		result, err := d.ListNotifications(ctx, nil, nil, 50, 0)
		if err != nil {
			t.Errorf("ListNotifications() error = %v", err)
		}
		if len(result.Notifications) != 1 {
			t.Errorf("ListNotifications() returned %d notifications, want 1", len(result.Notifications))
		}
		if result.Total != 1 {
			t.Errorf("ListNotifications() total = %d, want 1", result.Total)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})

	t.Run("list by client", func(t *testing.T) {
		clientID := "client-1"
		mock.ExpectQuery("SELECT COUNT").
			WithArgs(clientID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		rows := sqlmock.NewRows([]string{"notification_id", "client_id", "alert_id", "severity", "source", "name", "context", "rule_ids", "status", "created_at", "updated_at"}).
			AddRow("notif-1", "client-1", "alert-1", "HIGH", "source-1", "alert-1", nil, pq.Array([]string{"rule-1"}), "RECEIVED", time.Now(), time.Now())
		mock.ExpectQuery("SELECT notification_id, client_id, alert_id, severity, source, name, context, rule_ids, status, created_at, updated_at").
			WithArgs(clientID, 50, 0).
			WillReturnRows(rows)

		result, err := d.ListNotifications(ctx, &clientID, nil, 50, 0)
		if err != nil {
			t.Errorf("ListNotifications() error = %v", err)
		}
		if len(result.Notifications) != 1 {
			t.Errorf("ListNotifications() returned %d notifications, want 1", len(result.Notifications))
		}
		if result.Total != 1 {
			t.Errorf("ListNotifications() total = %d, want 1", result.Total)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})

	t.Run("list by status", func(t *testing.T) {
		status := "RECEIVED"
		mock.ExpectQuery("SELECT COUNT").
			WithArgs(status).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		rows := sqlmock.NewRows([]string{"notification_id", "client_id", "alert_id", "severity", "source", "name", "context", "rule_ids", "status", "created_at", "updated_at"}).
			AddRow("notif-1", "client-1", "alert-1", "HIGH", "source-1", "alert-1", nil, pq.Array([]string{"rule-1"}), "RECEIVED", time.Now(), time.Now())
		mock.ExpectQuery("SELECT notification_id, client_id, alert_id, severity, source, name, context, rule_ids, status, created_at, updated_at").
			WithArgs(status, 50, 0).
			WillReturnRows(rows)

		result, err := d.ListNotifications(ctx, nil, &status, 50, 0)
		if err != nil {
			t.Errorf("ListNotifications() error = %v", err)
		}
		if len(result.Notifications) != 1 {
			t.Errorf("ListNotifications() returned %d notifications, want 1", len(result.Notifications))
		}
		if result.Total != 1 {
			t.Errorf("ListNotifications() total = %d, want 1", result.Total)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})

	t.Run("list by client and status", func(t *testing.T) {
		clientID := "client-1"
		status := "RECEIVED"
		mock.ExpectQuery("SELECT COUNT").
			WithArgs(clientID, status).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		rows := sqlmock.NewRows([]string{"notification_id", "client_id", "alert_id", "severity", "source", "name", "context", "rule_ids", "status", "created_at", "updated_at"}).
			AddRow("notif-1", "client-1", "alert-1", "HIGH", "source-1", "alert-1", nil, pq.Array([]string{"rule-1"}), "RECEIVED", time.Now(), time.Now())
		mock.ExpectQuery("SELECT notification_id, client_id, alert_id, severity, source, name, context, rule_ids, status, created_at, updated_at").
			WithArgs(clientID, status, 50, 0).
			WillReturnRows(rows)

		result, err := d.ListNotifications(ctx, &clientID, &status, 50, 0)
		if err != nil {
			t.Errorf("ListNotifications() error = %v", err)
		}
		if len(result.Notifications) != 1 {
			t.Errorf("ListNotifications() returned %d notifications, want 1", len(result.Notifications))
		}
		if result.Total != 1 {
			t.Errorf("ListNotifications() total = %d, want 1", result.Total)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}

// Helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
