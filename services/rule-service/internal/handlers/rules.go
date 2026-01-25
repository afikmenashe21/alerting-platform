// Package handlers provides HTTP handlers for the rule-service API.
package handlers

import (
	"net/http"

	"rule-service/internal/events"
)

// CreateRuleRequest represents a request to create a rule.
type CreateRuleRequest struct {
	ClientID string `json:"client_id"`
	Severity string `json:"severity"`
	Source   string `json:"source"`
	Name     string `json:"name"`
}

// UpdateRuleRequest represents a request to update a rule.
type UpdateRuleRequest struct {
	Severity string `json:"severity"`
	Source   string `json:"source"`
	Name     string `json:"name"`
	Version  int    `json:"version"` // Optimistic locking version
}

// ToggleRuleEnabledRequest represents a request to toggle rule enabled status.
type ToggleRuleEnabledRequest struct {
	Enabled bool `json:"enabled"`
	Version int  `json:"version"` // Optimistic locking version
}

// CreateRule creates a new rule and publishes a rule.changed event.
func (h *Handlers) CreateRule(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var req CreateRuleRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if req.ClientID == "" {
		http.Error(w, "client_id is required", http.StatusBadRequest)
		return
	}

	if !validateRuleFields(w, req.Severity, req.Source, req.Name) {
		return
	}

	if !validateRuleValues(w, req.Severity, req.Source, req.Name) {
		return
	}

	ctx := r.Context()
	rule, err := h.db.CreateRule(ctx, req.ClientID, req.Severity, req.Source, req.Name)
	if err != nil {
		if handleDBError(w, err, "rule", req.ClientID) {
			return
		}
		http.Error(w, "Failed to create rule: "+err.Error(), http.StatusBadRequest)
		return
	}

	h.publishRuleChangedEvent(ctx, rule, events.ActionCreated)

	writeJSON(w, http.StatusCreated, rule)
}

// GetRule retrieves a rule by ID.
func (h *Handlers) GetRule(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	ruleID, ok := requireQueryParam(w, r, "rule_id")
	if !ok {
		return
	}

	ctx := r.Context()
	rule, err := h.db.GetRule(ctx, ruleID)
	if err != nil {
		if handleDBError(w, err, "rule", ruleID) {
			return
		}
		http.Error(w, "Failed to get rule: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, rule)
}

// ListRules retrieves rules with pagination, optionally filtered by client_id.
// Query params: client_id, limit (default 50, max 200), offset (default 0)
func (h *Handlers) ListRules(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	clientID := r.URL.Query().Get("client_id")
	var clientIDPtr *string
	if clientID != "" {
		clientIDPtr = &clientID
	}

	p := parsePagination(r)
	ctx := r.Context()
	result, err := h.db.ListRules(ctx, clientIDPtr, p.Limit, p.Offset)
	if err != nil {
		if handleDBError(w, err, "rule", "") {
			return
		}
		http.Error(w, "Failed to list rules: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// UpdateRule updates a rule and publishes a rule.changed event.
func (h *Handlers) UpdateRule(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPut) {
		return
	}

	ruleID, ok := requireQueryParam(w, r, "rule_id")
	if !ok {
		return
	}

	var req UpdateRuleRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if !validateRuleFields(w, req.Severity, req.Source, req.Name) {
		return
	}

	if !validateRuleValues(w, req.Severity, req.Source, req.Name) {
		return
	}

	ctx := r.Context()
	rule, err := h.db.UpdateRule(ctx, ruleID, req.Severity, req.Source, req.Name, req.Version)
	if err != nil {
		if handleDBError(w, err, "rule", ruleID) {
			return
		}
		http.Error(w, "Failed to update rule: "+err.Error(), http.StatusBadRequest)
		return
	}

	h.publishRuleChangedEvent(ctx, rule, events.ActionUpdated)

	writeJSON(w, http.StatusOK, rule)
}

// ToggleRuleEnabled toggles the enabled status of a rule and publishes a rule.changed event.
func (h *Handlers) ToggleRuleEnabled(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	ruleID, ok := requireQueryParam(w, r, "rule_id")
	if !ok {
		return
	}

	var req ToggleRuleEnabledRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	ctx := r.Context()
	rule, err := h.db.ToggleRuleEnabled(ctx, ruleID, req.Enabled, req.Version)
	if err != nil {
		if handleDBError(w, err, "rule", ruleID) {
			return
		}
		http.Error(w, "Failed to toggle rule enabled: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Determine action: DISABLED if disabling, UPDATED if re-enabling
	action := events.ActionDisabled
	if rule.Enabled {
		action = events.ActionUpdated // Re-enabling is treated as update
	}
	h.publishRuleChangedEvent(ctx, rule, action)

	writeJSON(w, http.StatusOK, rule)
}

// DeleteRule deletes a rule and publishes a rule.changed event.
func (h *Handlers) DeleteRule(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodDelete) {
		return
	}

	ruleID, ok := requireQueryParam(w, r, "rule_id")
	if !ok {
		return
	}

	ctx := r.Context()

	// Get rule before deletion to publish event
	rule, err := h.db.GetRule(ctx, ruleID)
	if err != nil {
		if handleDBError(w, err, "rule", ruleID) {
			return
		}
		http.Error(w, "Failed to get rule for deletion: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete the rule
	if err := h.db.DeleteRule(ctx, ruleID); err != nil {
		if handleDBError(w, err, "rule", ruleID) {
			return
		}
		http.Error(w, "Failed to delete rule: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Publish rule.changed event after successful DB commit
	h.publishRuleDeletedEvent(ctx, rule)

	w.WriteHeader(http.StatusNoContent)
}
