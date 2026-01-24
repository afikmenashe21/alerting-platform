// Package handlers provides HTTP handlers for the rule-service API.
package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
)

// CreateEndpointRequest represents a request to create an endpoint.
type CreateEndpointRequest struct {
	RuleID string `json:"rule_id"`
	Type   string `json:"type"`   // email, webhook, slack
	Value  string `json:"value"`  // email address, URL, etc.
}

// UpdateEndpointRequest represents a request to update an endpoint.
type UpdateEndpointRequest struct {
	Type  string `json:"type"`  // email, webhook, slack
	Value string `json:"value"` // email address, URL, etc.
}

// ToggleEndpointEnabledRequest represents a request to toggle endpoint enabled status.
type ToggleEndpointEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

// CreateEndpoint creates a new endpoint for a rule.
func (h *Handlers) CreateEndpoint(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var req CreateEndpointRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if req.RuleID == "" {
		http.Error(w, "rule_id is required", http.StatusBadRequest)
		return
	}
	if req.Type == "" {
		http.Error(w, "type is required", http.StatusBadRequest)
		return
	}
	if req.Value == "" {
		http.Error(w, "value is required", http.StatusBadRequest)
		return
	}

	// Validate endpoint type enum
	if !isValidEndpointType(req.Type) {
		http.Error(w, "type must be one of: email, webhook, slack", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	endpoint, err := h.db.CreateEndpoint(ctx, req.RuleID, req.Type, req.Value)
	if err != nil {
		if handleDBError(w, err, "endpoint", req.RuleID) {
			return
		}
		http.Error(w, "Failed to create endpoint: "+err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusCreated, endpoint)
}

// GetEndpoint retrieves an endpoint by ID.
func (h *Handlers) GetEndpoint(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	endpointID, ok := requireQueryParam(w, r, "endpoint_id")
	if !ok {
		return
	}

	ctx := r.Context()
	endpoint, err := h.db.GetEndpoint(ctx, endpointID)
	if err != nil {
		if handleDBError(w, err, "endpoint", endpointID) {
			return
		}
		http.Error(w, "Failed to get endpoint: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, endpoint)
}

// ListEndpoints retrieves endpoints with pagination, optionally filtered by rule_id.
// Query params: rule_id (optional), limit (default 50, max 200), offset (default 0)
func (h *Handlers) ListEndpoints(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	ruleID := r.URL.Query().Get("rule_id")
	var ruleIDPtr *string
	if ruleID != "" {
		ruleIDPtr = &ruleID
	}

	// Parse pagination parameters
	limit := 50 // default
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0 // default
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	ctx := r.Context()
	result, err := h.db.ListEndpoints(ctx, ruleIDPtr, limit, offset)
	if err != nil {
		slog.Error("Failed to list endpoints", "error", err, "rule_id", ruleID)
		http.Error(w, "Failed to list endpoints", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// UpdateEndpoint updates an endpoint.
func (h *Handlers) UpdateEndpoint(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPut) {
		return
	}

	endpointID, ok := requireQueryParam(w, r, "endpoint_id")
	if !ok {
		return
	}

	var req UpdateEndpointRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if req.Type == "" {
		http.Error(w, "type is required", http.StatusBadRequest)
		return
	}
	if req.Value == "" {
		http.Error(w, "value is required", http.StatusBadRequest)
		return
	}

	// Validate endpoint type enum
	if !isValidEndpointType(req.Type) {
		http.Error(w, "type must be one of: email, webhook, slack", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	endpoint, err := h.db.UpdateEndpoint(ctx, endpointID, req.Type, req.Value)
	if err != nil {
		if handleDBError(w, err, "endpoint", endpointID) {
			return
		}
		http.Error(w, "Failed to update endpoint: "+err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, endpoint)
}

// ToggleEndpointEnabled toggles the enabled status of an endpoint.
func (h *Handlers) ToggleEndpointEnabled(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	endpointID, ok := requireQueryParam(w, r, "endpoint_id")
	if !ok {
		return
	}

	var req ToggleEndpointEnabledRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	ctx := r.Context()
	endpoint, err := h.db.ToggleEndpointEnabled(ctx, endpointID, req.Enabled)
	if err != nil {
		if handleDBError(w, err, "endpoint", endpointID) {
			return
		}
		http.Error(w, "Failed to toggle endpoint enabled: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, endpoint)
}

// DeleteEndpoint deletes an endpoint.
func (h *Handlers) DeleteEndpoint(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodDelete) {
		return
	}

	endpointID, ok := requireQueryParam(w, r, "endpoint_id")
	if !ok {
		return
	}

	ctx := r.Context()
	if err := h.db.DeleteEndpoint(ctx, endpointID); err != nil {
		if handleDBError(w, err, "endpoint", endpointID) {
			return
		}
		http.Error(w, "Failed to delete endpoint: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
