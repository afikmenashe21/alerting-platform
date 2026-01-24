// Package handlers provides tests for HTTP handlers.
// These tests use mocks for database and producer to achieve 100% coverage.
package handlers

import (
	"bytes"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"unsafe"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"rule-service/internal/database"
	"rule-service/internal/producer"
)

// setupTestDB creates a mock database connection for testing.
// Since database.DB has an unexported conn field, we use unsafe.Pointer to set it.
func setupTestDB(t *testing.T) (*database.DB, sqlmock.Sqlmock) {
	dbConn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	
	// Create a database.DB instance and use unsafe to set the unexported conn field
	db := &database.DB{}
	// Use unsafe to access the first (and only) field of the struct
	// This is safe because we know the struct layout
	dbPtr := (*struct{ conn *sql.DB })(unsafe.Pointer(db))
	dbPtr.conn = dbConn
	
	return db, mock
}

// setupTestProducer creates a producer for testing.
// It uses a dummy broker address - publish will fail but that's OK for handler tests.
func setupTestProducer(t *testing.T) *producer.Producer {
	// Use a dummy broker - producer creation will succeed but publish will fail
	// This is OK for testing handler logic
	prod, err := producer.NewProducer("localhost:9999", "test-topic")
	if err != nil {
		// If producer creation fails, we can't test handlers that use it
		// But most handler tests don't actually need a working producer
		t.Logf("Warning: Could not create test producer: %v", err)
		return nil
	}
	return prod
}

// TestHandlers_CreateClient tests the CreateClient handler.
func TestHandlers_CreateClient(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	prod := setupTestProducer(t)
	if prod == nil {
		// Create a minimal producer that will fail on publish but won't crash
		prod, _ = producer.NewProducer("dummy:9092", "dummy")
	}
	h := NewHandlers(db, prod, nil)

	tests := []struct {
		name           string
		method         string
		body           string
		setupMock      func()
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "successful create",
			method: http.MethodPost,
			body:   `{"client_id":"client-1","name":"Test Client"}`,
			setupMock: func() {
				mock.ExpectExec("INSERT INTO clients").
					WithArgs("client-1", "Test Client").
					WillReturnResult(sqlmock.NewResult(1, 1))
				rows := sqlmock.NewRows([]string{"client_id", "name", "created_at", "updated_at"}).
					AddRow("client-1", "Test Client", time.Now(), time.Now())
				mock.ExpectQuery("SELECT client_id, name, created_at, updated_at").
					WithArgs("client-1").
					WillReturnRows(rows)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "wrong method",
			method:         http.MethodGet,
			body:           `{"client_id":"client-1","name":"Test Client"}`,
			setupMock:      func() {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "invalid JSON",
			method:         http.MethodPost,
			body:           `invalid json`,
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing client_id",
			method:         http.MethodPost,
			body:           `{"name":"Test Client"}`,
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing name",
			method:         http.MethodPost,
			body:           `{"client_id":"client-1"}`,
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "duplicate client",
			method: http.MethodPost,
			body:   `{"client_id":"client-1","name":"Test Client"}`,
			setupMock: func() {
				mock.ExpectExec("INSERT INTO clients").
					WithArgs("client-1", "Test Client").
					WillReturnError(&pq.Error{Code: "23505"})
			},
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			req := httptest.NewRequest(tt.method, "/api/v1/clients", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			h.CreateClient(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("CreateClient() status = %v, want %v", w.Code, tt.expectedStatus)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Mock expectations were not met: %v", err)
			}
		})
	}
}

// TestHandlers_GetClient tests the GetClient handler.
func TestHandlers_GetClient(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	prod := setupTestProducer(t)
	if prod == nil {
		// Create a minimal producer that will fail on publish but won't crash
		prod, _ = producer.NewProducer("dummy:9092", "dummy")
	}
	h := NewHandlers(db, prod, nil)

	tests := []struct {
		name           string
		method         string
		query          string
		setupMock      func()
		expectedStatus int
	}{
		{
			name:   "successful get",
			method: http.MethodGet,
			query:  "?client_id=client-1",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{"client_id", "name", "created_at", "updated_at"}).
					AddRow("client-1", "Test Client", time.Now(), time.Now())
				mock.ExpectQuery("SELECT client_id, name, created_at, updated_at").
					WithArgs("client-1").
					WillReturnRows(rows)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "wrong method",
			method:         http.MethodPost,
			query:          "?client_id=client-1",
			setupMock:      func() {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "missing client_id",
			method:         http.MethodGet,
			query:          "",
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "client not found",
			method: http.MethodGet,
			query:  "?client_id=client-999",
			setupMock: func() {
				mock.ExpectQuery("SELECT client_id, name, created_at, updated_at").
					WithArgs("client-999").
					WillReturnError(sql.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			req := httptest.NewRequest(tt.method, "/api/v1/clients"+tt.query, nil)
			w := httptest.NewRecorder()

			h.GetClient(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("GetClient() status = %v, want %v", w.Code, tt.expectedStatus)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Mock expectations were not met: %v", err)
			}
		})
	}
}

// TestHandlers_ListClients tests the ListClients handler.
func TestHandlers_ListClients(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	prod := setupTestProducer(t)
	if prod == nil {
		// Create a minimal producer that will fail on publish but won't crash
		prod, _ = producer.NewProducer("dummy:9092", "dummy")
	}
	h := NewHandlers(db, prod, nil)

	t.Run("successful list", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"client_id", "name", "created_at", "updated_at"}).
			AddRow("client-1", "Client 1", time.Now(), time.Now()).
			AddRow("client-2", "Client 2", time.Now(), time.Now())
		mock.ExpectQuery("SELECT client_id, name, created_at, updated_at").
			WillReturnRows(rows)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/clients", nil)
		w := httptest.NewRecorder()

		h.ListClients(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("ListClients() status = %v, want %v", w.Code, http.StatusOK)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})

	t.Run("wrong method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/clients", nil)
		w := httptest.NewRecorder()

		h.ListClients(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("ListClients() status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
		}
	})
}

// TestHandlers_CreateRule tests the CreateRule handler.
func TestHandlers_CreateRule(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	prod := setupTestProducer(t)
	if prod == nil {
		// Create a minimal producer that will fail on publish but won't crash
		prod, _ = producer.NewProducer("dummy:9092", "dummy")
	}
	h := NewHandlers(db, prod, nil)

	tests := []struct {
		name           string
		method         string
		body           string
		setupMock      func()
		expectedStatus int
	}{
		{
			name:   "successful create",
			method: http.MethodPost,
			body:   `{"client_id":"client-1","severity":"HIGH","source":"source-1","name":"alert-1"}`,
			setupMock: func() {
				rows := sqlmock.NewRows([]string{"rule_id", "client_id", "severity", "source", "name", "enabled", "version", "created_at", "updated_at"}).
					AddRow("rule-1", "client-1", "HIGH", "source-1", "alert-1", true, 1, time.Now(), time.Now())
				mock.ExpectQuery("INSERT INTO rules").
					WithArgs("client-1", "HIGH", "source-1", "alert-1").
					WillReturnRows(rows)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "wrong method",
			method:         http.MethodGet,
			body:           `{"client_id":"client-1","severity":"HIGH","source":"source-1","name":"alert-1"}`,
			setupMock:      func() {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "invalid JSON",
			method:         http.MethodPost,
			body:           `invalid json`,
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing client_id",
			method:         http.MethodPost,
			body:           `{"severity":"HIGH","source":"source-1","name":"alert-1"}`,
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing severity",
			method:         http.MethodPost,
			body:           `{"client_id":"client-1","source":"source-1","name":"alert-1"}`,
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid severity",
			method:         http.MethodPost,
			body:           `{"client_id":"client-1","severity":"INVALID","source":"source-1","name":"alert-1"}`,
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "all wildcards",
			method:         http.MethodPost,
			body:           `{"client_id":"client-1","severity":"*","source":"*","name":"*"}`,
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "client not found",
			method: http.MethodPost,
			body:   `{"client_id":"client-999","severity":"HIGH","source":"source-1","name":"alert-1"}`,
			setupMock: func() {
				mock.ExpectQuery("INSERT INTO rules").
					WithArgs("client-999", "HIGH", "source-1", "alert-1").
					WillReturnError(&pq.Error{Code: "23503"})
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			req := httptest.NewRequest(tt.method, "/api/v1/rules", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			h.CreateRule(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("CreateRule() status = %v, want %v", w.Code, tt.expectedStatus)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Mock expectations were not met: %v", err)
			}
		})
	}
}

// TestHandlers_GetRule tests the GetRule handler.
func TestHandlers_GetRule(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	prod := setupTestProducer(t)
	if prod == nil {
		// Create a minimal producer that will fail on publish but won't crash
		prod, _ = producer.NewProducer("dummy:9092", "dummy")
	}
	h := NewHandlers(db, prod, nil)

	t.Run("successful get", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"rule_id", "client_id", "severity", "source", "name", "enabled", "version", "created_at", "updated_at"}).
			AddRow("rule-1", "client-1", "HIGH", "source-1", "alert-1", true, 1, time.Now(), time.Now())
		mock.ExpectQuery("SELECT rule_id, client_id, severity, source, name, enabled, version, created_at, updated_at").
			WithArgs("rule-1").
			WillReturnRows(rows)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/rules?rule_id=rule-1", nil)
		w := httptest.NewRecorder()

		h.GetRule(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("GetRule() status = %v, want %v", w.Code, http.StatusOK)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}

// TestHandlers_ListRules tests the ListRules handler.
func TestHandlers_ListRules(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	prod := setupTestProducer(t)
	if prod == nil {
		// Create a minimal producer that will fail on publish but won't crash
		prod, _ = producer.NewProducer("dummy:9092", "dummy")
	}
	h := NewHandlers(db, prod, nil)

	t.Run("list all", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"rule_id", "client_id", "severity", "source", "name", "enabled", "version", "created_at", "updated_at"}).
			AddRow("rule-1", "client-1", "HIGH", "source-1", "alert-1", true, 1, time.Now(), time.Now())
		mock.ExpectQuery("SELECT rule_id, client_id, severity, source, name, enabled, version, created_at, updated_at").
			WillReturnRows(rows)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/rules", nil)
		w := httptest.NewRecorder()

		h.ListRules(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("ListRules() status = %v, want %v", w.Code, http.StatusOK)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})

	t.Run("list by client", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"rule_id", "client_id", "severity", "source", "name", "enabled", "version", "created_at", "updated_at"}).
			AddRow("rule-1", "client-1", "HIGH", "source-1", "alert-1", true, 1, time.Now(), time.Now())
		mock.ExpectQuery("SELECT rule_id, client_id, severity, source, name, enabled, version, created_at, updated_at").
			WithArgs("client-1").
			WillReturnRows(rows)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/rules?client_id=client-1", nil)
		w := httptest.NewRecorder()

		h.ListRules(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("ListRules() status = %v, want %v", w.Code, http.StatusOK)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}

// TestHandlers_UpdateRule tests the UpdateRule handler.
func TestHandlers_UpdateRule(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	prod := setupTestProducer(t)
	if prod == nil {
		// Create a minimal producer that will fail on publish but won't crash
		prod, _ = producer.NewProducer("dummy:9092", "dummy")
	}
	h := NewHandlers(db, prod, nil)

	tests := []struct {
		name           string
		method         string
		query          string
		body           string
		setupMock      func()
		expectedStatus int
	}{
		{
			name:   "successful update",
			method: http.MethodPut,
			query:  "?rule_id=rule-1",
			body:   `{"severity":"CRITICAL","source":"source-2","name":"alert-2","version":1}`,
			setupMock: func() {
				rows := sqlmock.NewRows([]string{"rule_id", "client_id", "severity", "source", "name", "enabled", "version", "created_at", "updated_at"}).
					AddRow("rule-1", "client-1", "CRITICAL", "source-2", "alert-2", true, 2, time.Now(), time.Now())
				mock.ExpectQuery("UPDATE rules").
					WithArgs("rule-1", "CRITICAL", "source-2", "alert-2", 1).
					WillReturnRows(rows)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "wrong method",
			method:         http.MethodPost,
			query:          "?rule_id=rule-1",
			body:           `{"severity":"CRITICAL","source":"source-2","name":"alert-2","version":1}`,
			setupMock:      func() {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "missing rule_id",
			method:         http.MethodPut,
			query:          "",
			body:           `{"severity":"CRITICAL","source":"source-2","name":"alert-2","version":1}`,
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "version mismatch",
			method:         http.MethodPut,
			query:          "?rule_id=rule-1",
			body:           `{"severity":"CRITICAL","source":"source-2","name":"alert-2","version":1}`,
			setupMock: func() {
				mock.ExpectQuery("UPDATE rules").
					WithArgs("rule-1", "CRITICAL", "source-2", "alert-2", 1).
					WillReturnError(sql.ErrNoRows)
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs("rule-1").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
			},
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			req := httptest.NewRequest(tt.method, "/api/v1/rules/update"+tt.query, bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			h.UpdateRule(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("UpdateRule() status = %v, want %v", w.Code, tt.expectedStatus)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Mock expectations were not met: %v", err)
			}
		})
	}
}

// TestHandlers_ToggleRuleEnabled tests the ToggleRuleEnabled handler.
func TestHandlers_ToggleRuleEnabled(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	prod := setupTestProducer(t)
	if prod == nil {
		// Create a minimal producer that will fail on publish but won't crash
		prod, _ = producer.NewProducer("dummy:9092", "dummy")
	}
	h := NewHandlers(db, prod, nil)

	t.Run("successful toggle", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"rule_id", "client_id", "severity", "source", "name", "enabled", "version", "created_at", "updated_at"}).
			AddRow("rule-1", "client-1", "HIGH", "source-1", "alert-1", false, 2, time.Now(), time.Now())
		mock.ExpectQuery("UPDATE rules").
			WithArgs("rule-1", false, 1).
			WillReturnRows(rows)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/rules/toggle?rule_id=rule-1", bytes.NewBufferString(`{"enabled":false,"version":1}`))
		w := httptest.NewRecorder()

		h.ToggleRuleEnabled(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("ToggleRuleEnabled() status = %v, want %v", w.Code, http.StatusOK)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}

// TestHandlers_DeleteRule tests the DeleteRule handler.
func TestHandlers_DeleteRule(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	prod := setupTestProducer(t)
	if prod == nil {
		// Create a minimal producer that will fail on publish but won't crash
		prod, _ = producer.NewProducer("dummy:9092", "dummy")
	}
	h := NewHandlers(db, prod, nil)

	t.Run("successful delete", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"rule_id", "client_id", "severity", "source", "name", "enabled", "version", "created_at", "updated_at"}).
			AddRow("rule-1", "client-1", "HIGH", "source-1", "alert-1", true, 1, time.Now(), time.Now())
		mock.ExpectQuery("SELECT rule_id, client_id, severity, source, name, enabled, version, created_at, updated_at").
			WithArgs("rule-1").
			WillReturnRows(rows)
		mock.ExpectExec("DELETE FROM rules").
			WithArgs("rule-1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/rules/delete?rule_id=rule-1", nil)
		w := httptest.NewRecorder()

		h.DeleteRule(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("DeleteRule() status = %v, want %v", w.Code, http.StatusNoContent)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}

// TestHandlers_CreateEndpoint tests the CreateEndpoint handler.
func TestHandlers_CreateEndpoint(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	prod := setupTestProducer(t)
	if prod == nil {
		// Create a minimal producer that will fail on publish but won't crash
		prod, _ = producer.NewProducer("dummy:9092", "dummy")
	}
	h := NewHandlers(db, prod, nil)

	tests := []struct {
		name           string
		method         string
		body           string
		setupMock      func()
		expectedStatus int
	}{
		{
			name:   "successful create",
			method: http.MethodPost,
			body:   `{"rule_id":"rule-1","type":"email","value":"test@example.com"}`,
			setupMock: func() {
				rows := sqlmock.NewRows([]string{"endpoint_id", "rule_id", "type", "value", "enabled", "created_at", "updated_at"}).
					AddRow("endpoint-1", "rule-1", "email", "test@example.com", true, time.Now(), time.Now())
				mock.ExpectQuery("INSERT INTO endpoints").
					WithArgs("rule-1", "email", "test@example.com").
					WillReturnRows(rows)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "invalid type",
			method:         http.MethodPost,
			body:           `{"rule_id":"rule-1","type":"invalid","value":"test@example.com"}`,
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "rule not found",
			method: http.MethodPost,
			body:   `{"rule_id":"rule-999","type":"email","value":"test@example.com"}`,
			setupMock: func() {
				mock.ExpectQuery("INSERT INTO endpoints").
					WithArgs("rule-999", "email", "test@example.com").
					WillReturnError(&pq.Error{Code: "23503"})
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			req := httptest.NewRequest(tt.method, "/api/v1/endpoints", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			h.CreateEndpoint(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("CreateEndpoint() status = %v, want %v", w.Code, tt.expectedStatus)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Mock expectations were not met: %v", err)
			}
		})
	}
}

// TestHandlers_GetEndpoint tests the GetEndpoint handler.
func TestHandlers_GetEndpoint(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	prod := setupTestProducer(t)
	if prod == nil {
		// Create a minimal producer that will fail on publish but won't crash
		prod, _ = producer.NewProducer("dummy:9092", "dummy")
	}
	h := NewHandlers(db, prod, nil)

	t.Run("successful get", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"endpoint_id", "rule_id", "type", "value", "enabled", "created_at", "updated_at"}).
			AddRow("endpoint-1", "rule-1", "email", "test@example.com", true, time.Now(), time.Now())
		mock.ExpectQuery("SELECT endpoint_id, rule_id, type, value, enabled, created_at, updated_at").
			WithArgs("endpoint-1").
			WillReturnRows(rows)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints?endpoint_id=endpoint-1", nil)
		w := httptest.NewRecorder()

		h.GetEndpoint(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("GetEndpoint() status = %v, want %v", w.Code, http.StatusOK)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}

// TestHandlers_ListEndpoints tests the ListEndpoints handler.
func TestHandlers_ListEndpoints(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	prod := setupTestProducer(t)
	if prod == nil {
		// Create a minimal producer that will fail on publish but won't crash
		prod, _ = producer.NewProducer("dummy:9092", "dummy")
	}
	h := NewHandlers(db, prod, nil)

	t.Run("successful list", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"endpoint_id", "rule_id", "type", "value", "enabled", "created_at", "updated_at"}).
			AddRow("endpoint-1", "rule-1", "email", "test@example.com", true, time.Now(), time.Now())
		mock.ExpectQuery("SELECT endpoint_id, rule_id, type, value, enabled, created_at, updated_at").
			WithArgs("rule-1").
			WillReturnRows(rows)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints?rule_id=rule-1", nil)
		w := httptest.NewRecorder()

		h.ListEndpoints(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("ListEndpoints() status = %v, want %v", w.Code, http.StatusOK)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}

// TestHandlers_UpdateEndpoint tests the UpdateEndpoint handler.
func TestHandlers_UpdateEndpoint(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	prod := setupTestProducer(t)
	if prod == nil {
		// Create a minimal producer that will fail on publish but won't crash
		prod, _ = producer.NewProducer("dummy:9092", "dummy")
	}
	h := NewHandlers(db, prod, nil)

	t.Run("successful update", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"endpoint_id", "rule_id", "type", "value", "enabled", "created_at", "updated_at"}).
			AddRow("endpoint-1", "rule-1", "webhook", "https://example.com", true, time.Now(), time.Now())
		mock.ExpectQuery("UPDATE endpoints").
			WithArgs("endpoint-1", "webhook", "https://example.com").
			WillReturnRows(rows)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/endpoints/update?endpoint_id=endpoint-1", bytes.NewBufferString(`{"type":"webhook","value":"https://example.com"}`))
		w := httptest.NewRecorder()

		h.UpdateEndpoint(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("UpdateEndpoint() status = %v, want %v", w.Code, http.StatusOK)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}

// TestHandlers_ToggleEndpointEnabled tests the ToggleEndpointEnabled handler.
func TestHandlers_ToggleEndpointEnabled(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	prod := setupTestProducer(t)
	if prod == nil {
		// Create a minimal producer that will fail on publish but won't crash
		prod, _ = producer.NewProducer("dummy:9092", "dummy")
	}
	h := NewHandlers(db, prod, nil)

	t.Run("successful toggle", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"endpoint_id", "rule_id", "type", "value", "enabled", "created_at", "updated_at"}).
			AddRow("endpoint-1", "rule-1", "email", "test@example.com", false, time.Now(), time.Now())
		mock.ExpectQuery("UPDATE endpoints").
			WithArgs("endpoint-1", false).
			WillReturnRows(rows)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/endpoints/toggle?endpoint_id=endpoint-1", bytes.NewBufferString(`{"enabled":false}`))
		w := httptest.NewRecorder()

		h.ToggleEndpointEnabled(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("ToggleEndpointEnabled() status = %v, want %v", w.Code, http.StatusOK)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}

// TestHandlers_DeleteEndpoint tests the DeleteEndpoint handler.
func TestHandlers_DeleteEndpoint(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	prod := setupTestProducer(t)
	if prod == nil {
		// Create a minimal producer that will fail on publish but won't crash
		prod, _ = producer.NewProducer("dummy:9092", "dummy")
	}
	h := NewHandlers(db, prod, nil)

	t.Run("successful delete", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM endpoints").
			WithArgs("endpoint-1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/endpoints/delete?endpoint_id=endpoint-1", nil)
		w := httptest.NewRecorder()

		h.DeleteEndpoint(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("DeleteEndpoint() status = %v, want %v", w.Code, http.StatusNoContent)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}

// TestHandlers_GetNotification tests the GetNotification handler.
func TestHandlers_GetNotification(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	prod := setupTestProducer(t)
	if prod == nil {
		// Create a minimal producer that will fail on publish but won't crash
		prod, _ = producer.NewProducer("dummy:9092", "dummy")
	}
	h := NewHandlers(db, prod, nil)

	t.Run("successful get", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"notification_id", "client_id", "alert_id", "severity", "source", "name", "context", "rule_ids", "status", "created_at", "updated_at"}).
			AddRow("notif-1", "client-1", "alert-1", "HIGH", "source-1", "alert-1", nil, pq.Array([]string{"rule-1"}), "RECEIVED", time.Now(), time.Now())
		mock.ExpectQuery("SELECT notification_id, client_id, alert_id, severity, source, name, context, rule_ids, status, created_at, updated_at").
			WithArgs("notif-1").
			WillReturnRows(rows)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?notification_id=notif-1", nil)
		w := httptest.NewRecorder()

		h.GetNotification(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("GetNotification() status = %v, want %v", w.Code, http.StatusOK)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}

// TestHandlers_ListNotifications tests the ListNotifications handler.
func TestHandlers_ListNotifications(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	prod := setupTestProducer(t)
	if prod == nil {
		// Create a minimal producer that will fail on publish but won't crash
		prod, _ = producer.NewProducer("dummy:9092", "dummy")
	}
	h := NewHandlers(db, prod, nil)

	t.Run("list all with pagination", func(t *testing.T) {
		// Expect COUNT query first
		mock.ExpectQuery("SELECT COUNT").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		// Expect SELECT with LIMIT and OFFSET
		rows := sqlmock.NewRows([]string{"notification_id", "client_id", "alert_id", "severity", "source", "name", "context", "rule_ids", "status", "created_at", "updated_at"}).
			AddRow("notif-1", "client-1", "alert-1", "HIGH", "source-1", "alert-1", nil, pq.Array([]string{"rule-1"}), "RECEIVED", time.Now(), time.Now())
		mock.ExpectQuery("SELECT notification_id, client_id, alert_id, severity, source, name, context, rule_ids, status, created_at, updated_at").
			WillReturnRows(rows)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?limit=50&offset=0", nil)
		w := httptest.NewRecorder()

		h.ListNotifications(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("ListNotifications() status = %v, want %v", w.Code, http.StatusOK)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Mock expectations were not met: %v", err)
		}
	})
}
