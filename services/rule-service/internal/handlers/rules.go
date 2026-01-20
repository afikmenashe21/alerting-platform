// Package handlers provides HTTP handlers for the rule-service API.
package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"rule-service/internal/database"
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

// publishRuleChangedEvent publishes a rule.changed event after a successful DB operation.
// It logs errors but does not fail the operation if publishing fails.
func (h *Handlers) publishRuleChangedEvent(ctx context.Context, rule *database.Rule, action string) {
	changed := &events.RuleChanged{
		RuleID:        rule.RuleID,
		ClientID:      rule.ClientID,
		Action:        action,
		Version:       rule.Version,
		UpdatedAt:     rule.UpdatedAt.Unix(),
		SchemaVersion: SchemaVersion,
	}
	if err := h.producer.Publish(ctx, changed); err != nil {
		slog.Error("Failed to publish rule.changed event", "error", err, "rule_id", rule.RuleID)
		// Continue - the rule operation succeeded, event publishing failure can be handled separately
	}
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
	if req.Severity == "" {
		http.Error(w, "severity is required", http.StatusBadRequest)
		return
	}
	if req.Source == "" {
		http.Error(w, "source is required", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	// Validate severity enum (allow "*" as wildcard)
	if !isValidSeverity(req.Severity) {
		http.Error(w, "severity must be one of: LOW, MEDIUM, HIGH, CRITICAL, or * (wildcard)", http.StatusBadRequest)
		return
	}

	// Validate that not all fields are wildcards
	if isAllWildcards(req.Severity, req.Source, req.Name) {
		http.Error(w, "cannot create rule with all fields as wildcards (*)", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	rule, err := h.db.CreateRule(ctx, req.ClientID, req.Severity, req.Source, req.Name)
	if err != nil {
		slog.Error("Failed to create rule", "error", err, "client_id", req.ClientID)
		if err.Error() == "client not found: "+req.ClientID {
			http.Error(w, "Client not found", http.StatusNotFound)
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
		slog.Error("Failed to get rule", "error", err, "rule_id", ruleID)
		http.Error(w, "Rule not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, rule)
}

// ListRules retrieves all rules, optionally filtered by client_id.
func (h *Handlers) ListRules(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	clientID := r.URL.Query().Get("client_id")
	var clientIDPtr *string
	if clientID != "" {
		clientIDPtr = &clientID
	}

	ctx := r.Context()
	rules, err := h.db.ListRules(ctx, clientIDPtr)
	if err != nil {
		slog.Error("Failed to list rules", "error", err)
		http.Error(w, "Failed to list rules", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, rules)
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

	if req.Severity == "" {
		http.Error(w, "severity is required", http.StatusBadRequest)
		return
	}
	if req.Source == "" {
		http.Error(w, "source is required", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	// Validate severity enum (allow "*" as wildcard)
	if !isValidSeverity(req.Severity) {
		http.Error(w, "severity must be one of: LOW, MEDIUM, HIGH, CRITICAL, or * (wildcard)", http.StatusBadRequest)
		return
	}

	// Validate that not all fields are wildcards
	if isAllWildcards(req.Severity, req.Source, req.Name) {
		http.Error(w, "cannot create rule with all fields as wildcards (*)", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	rule, err := h.db.UpdateRule(ctx, ruleID, req.Severity, req.Source, req.Name, req.Version)
	if err != nil {
		slog.Error("Failed to update rule", "error", err, "rule_id", ruleID)
		if err.Error() == "rule not found: "+ruleID {
			http.Error(w, "Rule not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "version mismatch") {
			http.Error(w, err.Error(), http.StatusConflict)
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
		slog.Error("Failed to toggle rule enabled", "error", err, "rule_id", ruleID)
		if err.Error() == "rule not found: "+ruleID {
			http.Error(w, "Rule not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "version mismatch") {
			http.Error(w, err.Error(), http.StatusConflict)
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
		slog.Error("Failed to get rule for deletion", "error", err, "rule_id", ruleID)
		http.Error(w, "Rule not found", http.StatusNotFound)
		return
	}

	// Delete the rule
	if err := h.db.DeleteRule(ctx, ruleID); err != nil {
		slog.Error("Failed to delete rule", "error", err, "rule_id", ruleID)
		http.Error(w, "Failed to delete rule", http.StatusInternalServerError)
		return
	}

	// Publish rule.changed event after successful DB commit
	// Use current time since rule.UpdatedAt may be stale after deletion
	changed := &events.RuleChanged{
		RuleID:        rule.RuleID,
		ClientID:      rule.ClientID,
		Action:        events.ActionDeleted,
		Version:       rule.Version,
		UpdatedAt:     time.Now().Unix(),
		SchemaVersion: SchemaVersion,
	}
	if err := h.producer.Publish(ctx, changed); err != nil {
		slog.Error("Failed to publish rule.changed event", "error", err, "rule_id", ruleID)
		// Continue - the rule was deleted, event publishing failure can be handled separately
	}

	w.WriteHeader(http.StatusNoContent)
}
