// Package handlers provides HTTP handlers for the rule-service API.
package handlers

import (
	"log/slog"
	"net/http"
)

// CreateClientRequest represents a request to create a client.
type CreateClientRequest struct {
	ClientID string `json:"client_id"`
	Name     string `json:"name"`
}

// CreateClient creates a new client.
func (h *Handlers) CreateClient(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var req CreateClientRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if req.ClientID == "" {
		http.Error(w, "client_id is required", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if err := h.db.CreateClient(ctx, req.ClientID, req.Name); err != nil {
		if handleDBError(w, err, "client", req.ClientID) {
			return
		}
		http.Error(w, "Failed to create client: "+err.Error(), http.StatusInternalServerError)
		return
	}

	client, err := h.db.GetClient(ctx, req.ClientID)
	if err != nil {
		slog.Error("Failed to get created client", "error", err, "client_id", req.ClientID)
		http.Error(w, "Failed to retrieve created client", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, client)
}

// GetClient retrieves a client by ID.
func (h *Handlers) GetClient(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	clientID, ok := requireQueryParam(w, r, "client_id")
	if !ok {
		return
	}

	ctx := r.Context()
	client, err := h.db.GetClient(ctx, clientID)
	if err != nil {
		if handleDBError(w, err, "client", clientID) {
			return
		}
		http.Error(w, "Failed to get client: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, client)
}

// ListClients retrieves all clients.
func (h *Handlers) ListClients(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	ctx := r.Context()
	clients, err := h.db.ListClients(ctx)
	if err != nil {
		slog.Error("Failed to list clients", "error", err)
		http.Error(w, "Failed to list clients", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, clients)
}
