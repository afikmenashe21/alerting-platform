// Package router provides tests for HTTP routing configuration.
package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"rule-service/internal/database"
	"rule-service/internal/handlers"
	"rule-service/internal/producer"
)

// TestNewRouter tests the NewRouter constructor.
func TestNewRouter(t *testing.T) {
	db := &database.DB{}
	prod := &producer.Producer{}
	h := handlers.NewHandlers(db, prod)

	router := NewRouter(h)
	if router == nil {
		t.Fatal("NewRouter() returned nil")
	}
	if router.mux == nil {
		t.Error("NewRouter() mux is nil")
	}
	if router.handlers != h {
		t.Error("NewRouter() handlers mismatch")
	}
}

// TestRouter_Handler tests that the router returns a handler with CORS middleware.
func TestRouter_Handler(t *testing.T) {
	db := &database.DB{}
	prod := &producer.Producer{}
	h := handlers.NewHandlers(db, prod)

	router := NewRouter(h)
	handler := router.Handler()
	if handler == nil {
		t.Fatal("Handler() returned nil")
	}

	// Test that CORS middleware is applied
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/clients", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("CORS OPTIONS request status = %v, want %v", w.Code, http.StatusOK)
	}

	// Check CORS headers
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("CORS header Access-Control-Allow-Origin not set")
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("CORS header Access-Control-Allow-Methods not set")
	}
}

// TestRouter_HealthCheck tests the health check endpoint.
func TestRouter_HealthCheck(t *testing.T) {
	db := &database.DB{}
	prod := &producer.Producer{}
	h := handlers.NewHandlers(db, prod)

	router := NewRouter(h)
	handler := router.Handler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Health check status = %v, want %v", w.Code, http.StatusOK)
	}
	if w.Body.String() != "OK" {
		t.Errorf("Health check body = %v, want OK", w.Body.String())
	}
}

// TestNewServer tests the NewServer constructor.
func TestNewServer(t *testing.T) {
	db := &database.DB{}
	prod := &producer.Producer{}
	h := handlers.NewHandlers(db, prod)

	server := NewServer("8081", h)
	if server == nil {
		t.Fatal("NewServer() returned nil")
	}
	if server.Addr != ":8081" {
		t.Errorf("NewServer() Addr = %v, want :8081", server.Addr)
	}
	if server.Handler == nil {
		t.Error("NewServer() Handler is nil")
	}
}

// TestRouter_Routes tests that routes are properly configured.
// This test verifies routes are registered by checking they don't return 404.
// Note: Some routes may panic or return errors due to nil database/producer, 
// but we're only checking that routes exist (not 404).
func TestRouter_Routes(t *testing.T) {
	// Create handlers with nil database/producer - routes will error but not 404
	db, _ := database.NewDB("postgres://invalid")
	if db != nil {
		defer db.Close()
	}
	prod, _ := producer.NewProducer("dummy:9092", "dummy")
	if prod != nil {
		defer prod.Close()
	}
	
	// If we can't create db/prod, use nil - test will still verify routes exist
	var h *handlers.Handlers
	if db != nil && prod != nil {
		h = handlers.NewHandlers(db, prod)
	} else {
		// Use nil - routes will panic but we catch it
		h = handlers.NewHandlers(nil, nil)
	}

	router := NewRouter(h)
	handler := router.Handler()

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"clients POST", http.MethodPost, "/api/v1/clients"},
		{"clients GET", http.MethodGet, "/api/v1/clients?client_id=test"},
		{"rules POST", http.MethodPost, "/api/v1/rules"},
		{"rules GET", http.MethodGet, "/api/v1/rules?rule_id=test"},
		{"rules UPDATE", http.MethodPut, "/api/v1/rules/update?rule_id=test"},
		{"rules TOGGLE", http.MethodPost, "/api/v1/rules/toggle?rule_id=test"},
		{"rules DELETE", http.MethodDelete, "/api/v1/rules/delete?rule_id=test"},
		{"endpoints POST", http.MethodPost, "/api/v1/endpoints"},
		{"endpoints GET", http.MethodGet, "/api/v1/endpoints?endpoint_id=test"},
		{"endpoints UPDATE", http.MethodPut, "/api/v1/endpoints/update?endpoint_id=test"},
		{"endpoints TOGGLE", http.MethodPost, "/api/v1/endpoints/toggle?endpoint_id=test"},
		{"endpoints DELETE", http.MethodDelete, "/api/v1/endpoints/delete?endpoint_id=test"},
		{"notifications GET", http.MethodGet, "/api/v1/notifications?notification_id=test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			// Catch panics - if route exists but panics due to nil, that's OK
			// We just want to verify the route is registered (not 404)
			panicked := false
			func() {
				defer func() {
					if r := recover(); r != nil {
						panicked = true
					}
				}()
				handler.ServeHTTP(w, req)
			}()

			// Routes should not return 404 (they exist)
			// Panics or other errors are OK - we're just checking route registration
			if !panicked && w.Code == http.StatusNotFound {
				t.Errorf("Route %s %s returned 404, route may not be registered", tt.method, tt.path)
			}
		})
	}
}

// TestCorsMiddleware tests CORS middleware functionality.
func TestCorsMiddleware(t *testing.T) {
	db := &database.DB{}
	prod := &producer.Producer{}
	h := handlers.NewHandlers(db, prod)

	router := NewRouter(h)
	handler := router.Handler()

	tests := []struct {
		name           string
		method         string
		expectedOrigin string
	}{
		{"GET request", http.MethodGet, "*"},
		{"POST request", http.MethodPost, "*"},
		{"PUT request", http.MethodPut, "*"},
		{"DELETE request", http.MethodDelete, "*"},
		{"OPTIONS request", http.MethodOptions, "*"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/health", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			origin := w.Header().Get("Access-Control-Allow-Origin")
			if origin != tt.expectedOrigin {
				t.Errorf("CORS Origin header = %v, want %v", origin, tt.expectedOrigin)
			}

			methods := w.Header().Get("Access-Control-Allow-Methods")
			if methods == "" {
				t.Error("CORS Methods header not set")
			}

			headers := w.Header().Get("Access-Control-Allow-Headers")
			if headers == "" {
				t.Error("CORS Headers header not set")
			}
		})
	}
}
