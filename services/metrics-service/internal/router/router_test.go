// Package router provides tests for HTTP routing configuration.
package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"metrics-service/internal/database"
	"metrics-service/internal/handlers"
)

// TestNewRouter tests the NewRouter constructor.
func TestNewRouter(t *testing.T) {
	db := &database.DB{}
	h := handlers.NewHandlers(db, nil, nil)

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
	h := handlers.NewHandlers(db, nil, nil)

	router := NewRouter(h)
	handler := router.Handler()
	if handler == nil {
		t.Fatal("Handler() returned nil")
	}

	// Test that CORS middleware is applied
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("CORS OPTIONS request status = %v, want %v", w.Code, http.StatusOK)
	}

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
	h := handlers.NewHandlers(db, nil, nil)

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
	h := handlers.NewHandlers(db, nil, nil)

	server := NewServer("8083", h)
	if server == nil {
		t.Fatal("NewServer() returned nil")
	}
	if server.Addr != ":8083" {
		t.Errorf("NewServer() Addr = %v, want :8083", server.Addr)
	}
	if server.Handler == nil {
		t.Error("NewServer() Handler is nil")
	}
}

// TestRouter_MethodNotAllowed tests that non-GET methods return 405.
func TestRouter_MethodNotAllowed(t *testing.T) {
	db := &database.DB{}
	h := handlers.NewHandlers(db, nil, nil)

	router := NewRouter(h)
	handler := router.Handler()

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"metrics POST", http.MethodPost, "/api/v1/metrics"},
		{"metrics PUT", http.MethodPut, "/api/v1/metrics"},
		{"metrics DELETE", http.MethodDelete, "/api/v1/metrics"},
		{"services/metrics POST", http.MethodPost, "/api/v1/services/metrics"},
		{"services/metrics PUT", http.MethodPut, "/api/v1/services/metrics"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Route %s %s status = %v, want %v", tt.method, tt.path, w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

// TestCorsMiddleware tests CORS middleware functionality.
func TestCorsMiddleware(t *testing.T) {
	db := &database.DB{}
	h := handlers.NewHandlers(db, nil, nil)

	router := NewRouter(h)
	handler := router.Handler()

	tests := []struct {
		name           string
		method         string
		expectedOrigin string
	}{
		{"GET request", http.MethodGet, "*"},
		{"POST request", http.MethodPost, "*"},
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
