// Package router provides HTTP routing configuration for the rule-service API.
package router

import (
	"net/http"
)

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

	// System metrics endpoint (database aggregates)
	r.mux.HandleFunc("/api/v1/metrics", func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodGet {
			r.handlers.GetSystemMetrics(w, req)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Service metrics endpoint (from Redis)
	r.mux.HandleFunc("/api/v1/services/metrics", func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodGet {
			r.handlers.GetServiceMetrics(w, req)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Health check endpoint
	r.mux.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
}
