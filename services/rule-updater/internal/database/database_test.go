package database

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

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
			}
			if db != nil {
				db.Close()
			}
		})
	}
}

func TestDB_Close(t *testing.T) {
	// Test Close with nil connection
	db := &DB{conn: nil}
	if err := db.Close(); err != nil {
		t.Errorf("DB.Close() with nil conn should not return error, got %v", err)
	}

	// Test Close with actual connection using sqlmock
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer mockDB.Close()

	mock.ExpectClose()

	db = &DB{conn: mockDB}
	if err := db.Close(); err != nil {
		t.Errorf("DB.Close() error = %v, want nil", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestDB_GetAllEnabledRules(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer mockDB.Close()

	db := &DB{conn: mockDB}
	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
		wantLen int
	}{
		{
			name: "success with rules",
			setup: func() {
				rows := sqlmock.NewRows([]string{"rule_id", "client_id", "severity", "source", "name", "enabled", "version", "created_at", "updated_at"}).
					AddRow("rule-1", "client-1", "HIGH", "source-1", "name-1", true, 1, time.Now(), time.Now()).
					AddRow("rule-2", "client-2", "MEDIUM", "source-2", "name-2", true, 1, time.Now(), time.Now())
				mock.ExpectQuery(`SELECT rule_id, client_id, severity, source, name, enabled, version, created_at, updated_at`).
					WillReturnRows(rows)
			},
			wantErr: false,
			wantLen: 2,
		},
		{
			name: "success with no rules",
			setup: func() {
				rows := sqlmock.NewRows([]string{"rule_id", "client_id", "severity", "source", "name", "enabled", "version", "created_at", "updated_at"})
				mock.ExpectQuery(`SELECT rule_id, client_id, severity, source, name, enabled, version, created_at, updated_at`).
					WillReturnRows(rows)
			},
			wantErr: false,
			wantLen: 0,
		},
		{
			name: "database error",
			setup: func() {
				mock.ExpectQuery(`SELECT rule_id, client_id, severity, source, name, enabled, version, created_at, updated_at`).
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			rules, err := db.GetAllEnabledRules(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAllEnabledRules() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(rules) != tt.wantLen {
				t.Errorf("GetAllEnabledRules() len = %v, want %v", len(rules), tt.wantLen)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestDB_GetRule(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer mockDB.Close()

	db := &DB{conn: mockDB}
	ctx := context.Background()

	tests := []struct {
		name    string
		ruleID  string
		setup   func()
		wantErr bool
		wantID  string
	}{
		{
			name:   "success",
			ruleID: "rule-1",
			setup: func() {
				rows := sqlmock.NewRows([]string{"rule_id", "client_id", "severity", "source", "name", "enabled", "version", "created_at", "updated_at"}).
					AddRow("rule-1", "client-1", "HIGH", "source-1", "name-1", true, 1, time.Now(), time.Now())
				mock.ExpectQuery(`SELECT rule_id, client_id, severity, source, name, enabled, version, created_at, updated_at`).
					WithArgs("rule-1").
					WillReturnRows(rows)
			},
			wantErr: false,
			wantID:  "rule-1",
		},
		{
			name:   "rule not found",
			ruleID: "rule-not-found",
			setup: func() {
				mock.ExpectQuery(`SELECT rule_id, client_id, severity, source, name, enabled, version, created_at, updated_at`).
					WithArgs("rule-not-found").
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			wantID:  "",
		},
		{
			name:   "database error",
			ruleID: "rule-1",
			setup: func() {
				mock.ExpectQuery(`SELECT rule_id, client_id, severity, source, name, enabled, version, created_at, updated_at`).
					WithArgs("rule-1").
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
			wantID:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			rule, err := db.GetRule(ctx, tt.ruleID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && rule != nil {
				if rule.RuleID != tt.wantID {
					t.Errorf("GetRule() RuleID = %v, want %v", rule.RuleID, tt.wantID)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestDB_GetAllEnabledRules_ScanError(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer mockDB.Close()

	db := &DB{conn: mockDB}
	ctx := context.Background()

	// Test scan error by providing wrong number of columns
	rows := sqlmock.NewRows([]string{"rule_id", "client_id"}).
		AddRow("rule-1", "client-1")
	mock.ExpectQuery(`SELECT rule_id, client_id, severity, source, name, enabled, version, created_at, updated_at`).
		WillReturnRows(rows)

	_, err = db.GetAllEnabledRules(ctx)
	if err == nil {
		t.Error("GetAllEnabledRules() expected error on scan, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}
