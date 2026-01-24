// Package handlers provides tests for HTTP handlers.
package handlers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"unsafe"

	"github.com/DATA-DOG/go-sqlmock"
	"metrics-service/internal/database"
)

// setupTestDB creates a mock database connection for testing.
func setupTestDB(t *testing.T) (*database.DB, sqlmock.Sqlmock) {
	dbConn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}

	db := &database.DB{}
	dbPtr := (*struct{ conn *sql.DB })(unsafe.Pointer(db))
	dbPtr.conn = dbConn

	return db, mock
}

// TestHandlers_GetSystemMetrics tests the GetSystemMetrics handler.
func TestHandlers_GetSystemMetrics(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	h := NewHandlers(db, nil, nil)

	t.Run("successful get", func(t *testing.T) {
		// Mock notification status query
		statusRows := sqlmock.NewRows([]string{"status", "count"}).
			AddRow("SENT", int64(10)).
			AddRow("RECEIVED", int64(5))
		mock.ExpectQuery("SELECT status, COUNT").WillReturnRows(statusRows)

		// Mock last 24h query
		mock.ExpectQuery("SELECT COUNT.*FROM notifications.*24 hours").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(8)))

		// Mock last hour query
		mock.ExpectQuery("SELECT COUNT.*FROM notifications.*1 hour").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(2)))

		// Mock hourly query
		hourlyRows := sqlmock.NewRows([]string{"hour", "count"}).
			AddRow(time.Now().Truncate(time.Hour), int64(3))
		mock.ExpectQuery("SELECT.*date_trunc").WillReturnRows(hourlyRows)

		// Mock rules query
		mock.ExpectQuery("SELECT.*COUNT.*FROM rules").
			WillReturnRows(sqlmock.NewRows([]string{"total", "enabled", "disabled"}).AddRow(int64(5), int64(3), int64(2)))

		// Mock clients query
		mock.ExpectQuery("SELECT COUNT.*FROM clients").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(2)))

		// Mock endpoints query
		endpointRows := sqlmock.NewRows([]string{"type", "count", "enabled_count"}).
			AddRow("email", int64(3), int64(2)).
			AddRow("webhook", int64(1), int64(1))
		mock.ExpectQuery("SELECT.*type.*COUNT.*FROM endpoints").WillReturnRows(endpointRows)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics", nil)
		w := httptest.NewRecorder()

		h.GetSystemMetrics(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("GetSystemMetrics() status = %v, want %v", w.Code, http.StatusOK)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery("SELECT status, COUNT").
			WillReturnError(sql.ErrConnDone)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics", nil)
		w := httptest.NewRecorder()

		h.GetSystemMetrics(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("GetSystemMetrics() status = %v, want %v", w.Code, http.StatusInternalServerError)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}

// TestHandlers_GetServiceMetrics tests the GetServiceMetrics handler.
func TestHandlers_GetServiceMetrics(t *testing.T) {
	h := NewHandlers(nil, nil, nil)

	t.Run("no reader returns error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/services/metrics", nil)
		w := httptest.NewRecorder()

		h.GetServiceMetrics(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("GetServiceMetrics() status = %v, want %v", w.Code, http.StatusInternalServerError)
		}
	})

	t.Run("specific service with no reader returns error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/services/metrics?service=evaluator", nil)
		w := httptest.NewRecorder()

		h.GetServiceMetrics(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("GetServiceMetrics() status = %v, want %v", w.Code, http.StatusInternalServerError)
		}
	})
}

// TestNewHandlers tests the NewHandlers constructor.
func TestNewHandlers(t *testing.T) {
	db := &database.DB{}

	h := NewHandlers(db, nil, nil)
	if h == nil {
		t.Fatal("NewHandlers() returned nil")
	}
	if h.db != db {
		t.Errorf("NewHandlers() db = %v, want %v", h.db, db)
	}
}
