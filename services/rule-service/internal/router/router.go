// Package router provides HTTP routing configuration for the rule-service API.
// It sets up routes and applies middleware like CORS.
package router

import (
	"net/http"
	"time"

	"rule-service/internal/handlers"
)

// Router wraps the HTTP mux and provides route configuration.
type Router struct {
	mux      *http.ServeMux
	handlers *handlers.Handlers
}

// NewRouter creates a new router with all routes configured.
func NewRouter(h *handlers.Handlers) *Router {
	r := &Router{
		mux:      http.NewServeMux(),
		handlers: h,
	}
	r.setupRoutes()
	return r
}

// setupRoutes configures all HTTP routes for the API.
func (r *Router) setupRoutes() {
	// Client endpoints
	r.mux.HandleFunc("/api/v1/clients", func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodPost:
			r.handlers.CreateClient(w, req)
		case http.MethodGet:
			if req.URL.Query().Get("client_id") != "" {
				r.handlers.GetClient(w, req)
			} else {
				r.handlers.ListClients(w, req)
			}
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Rule endpoints
	r.mux.HandleFunc("/api/v1/rules", func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodPost:
			r.handlers.CreateRule(w, req)
		case http.MethodGet:
			if req.URL.Query().Get("rule_id") != "" {
				r.handlers.GetRule(w, req)
			} else {
				r.handlers.ListRules(w, req)
			}
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	r.mux.HandleFunc("/api/v1/rules/update", func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPut {
			r.handlers.UpdateRule(w, req)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	r.mux.HandleFunc("/api/v1/rules/toggle", func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost {
			r.handlers.ToggleRuleEnabled(w, req)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	r.mux.HandleFunc("/api/v1/rules/delete", func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodDelete {
			r.handlers.DeleteRule(w, req)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Endpoint endpoints
	r.mux.HandleFunc("/api/v1/endpoints", func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodPost:
			r.handlers.CreateEndpoint(w, req)
		case http.MethodGet:
			if req.URL.Query().Get("endpoint_id") != "" {
				r.handlers.GetEndpoint(w, req)
			} else if req.URL.Query().Get("rule_id") != "" {
				r.handlers.ListEndpoints(w, req)
			} else {
				http.Error(w, "endpoint_id or rule_id query parameter is required", http.StatusBadRequest)
			}
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	r.mux.HandleFunc("/api/v1/endpoints/update", func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPut {
			r.handlers.UpdateEndpoint(w, req)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	r.mux.HandleFunc("/api/v1/endpoints/toggle", func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost {
			r.handlers.ToggleEndpointEnabled(w, req)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	r.mux.HandleFunc("/api/v1/endpoints/delete", func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodDelete {
			r.handlers.DeleteEndpoint(w, req)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Notification endpoints
	r.mux.HandleFunc("/api/v1/notifications", func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			if req.URL.Query().Get("notification_id") != "" {
				r.handlers.GetNotification(w, req)
			} else {
				r.handlers.ListNotifications(w, req)
			}
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Health check endpoint
	r.mux.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
}

// Handler returns the HTTP handler with CORS middleware applied.
func (r *Router) Handler() http.Handler {
	return corsMiddleware(r.mux)
}

// corsMiddleware applies CORS headers to all requests.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// NewServer creates a new HTTP server with the router configured.
func NewServer(port string, h *handlers.Handlers) *http.Server {
	router := NewRouter(h)
	return &http.Server{
		Addr:         ":" + port,
		Handler:      router.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}
