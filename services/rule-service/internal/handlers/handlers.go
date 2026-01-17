// Package handlers provides HTTP handlers for the rule-service API.
package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"rule-service/internal/events"
)

// CreateClientRequest represents a request to create a client.
type CreateClientRequest struct {
	ClientID string `json:"client_id"`
	Name     string `json:"name"`
}

// CreateClient creates a new client.
func (h *Handlers) CreateClient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
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
		slog.Error("Failed to create client", "error", err, "client_id", req.ClientID)
		if strings.Contains(err.Error(), "already exists") {
			http.Error(w, "Client already exists", http.StatusConflict)
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(client)
}

// GetClient retrieves a client by ID.
func (h *Handlers) GetClient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		http.Error(w, "client_id query parameter is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	client, err := h.db.GetClient(ctx, clientID)
	if err != nil {
		slog.Error("Failed to get client", "error", err, "client_id", clientID)
		http.Error(w, "Client not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(client)
}

// ListClients retrieves all clients.
func (h *Handlers) ListClients(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	clients, err := h.db.ListClients(ctx)
	if err != nil {
		slog.Error("Failed to list clients", "error", err)
		http.Error(w, "Failed to list clients", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clients)
}

// CreateRuleRequest represents a request to create a rule.
type CreateRuleRequest struct {
	ClientID string `json:"client_id"`
	Severity string `json:"severity"`
	Source   string `json:"source"`
	Name     string `json:"name"`
}

// CreateRule creates a new rule and publishes a rule.changed event.
func (h *Handlers) CreateRule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
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
	validSeverities := map[string]bool{"LOW": true, "MEDIUM": true, "HIGH": true, "CRITICAL": true, "*": true}
	if !validSeverities[req.Severity] {
		http.Error(w, "severity must be one of: LOW, MEDIUM, HIGH, CRITICAL, or * (wildcard)", http.StatusBadRequest)
		return
	}

	// Validate that not all fields are wildcards
	if req.Severity == "*" && req.Source == "*" && req.Name == "*" {
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

	// Publish rule.changed event after successful DB commit
	changed := &events.RuleChanged{
		RuleID:        rule.RuleID,
		ClientID:      rule.ClientID,
		Action:        events.ActionCreated,
		Version:       rule.Version,
		UpdatedAt:     rule.UpdatedAt.Unix(),
		SchemaVersion: SchemaVersion,
	}
	if err := h.producer.Publish(ctx, changed); err != nil {
		slog.Error("Failed to publish rule.changed event", "error", err, "rule_id", rule.RuleID)
		// Continue - the rule was created, event publishing failure can be handled separately
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rule)
}

// GetRule retrieves a rule by ID.
func (h *Handlers) GetRule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ruleID := r.URL.Query().Get("rule_id")
	if ruleID == "" {
		http.Error(w, "rule_id query parameter is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	rule, err := h.db.GetRule(ctx, ruleID)
	if err != nil {
		slog.Error("Failed to get rule", "error", err, "rule_id", ruleID)
		http.Error(w, "Rule not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rule)
}

// ListRules retrieves all rules, optionally filtered by client_id.
func (h *Handlers) ListRules(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}

// UpdateRuleRequest represents a request to update a rule.
type UpdateRuleRequest struct {
	Severity string `json:"severity"`
	Source   string `json:"source"`
	Name     string `json:"name"`
	Version  int    `json:"version"` // Optimistic locking version
}

// UpdateRule updates a rule and publishes a rule.changed event.
func (h *Handlers) UpdateRule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ruleID := r.URL.Query().Get("rule_id")
	if ruleID == "" {
		http.Error(w, "rule_id query parameter is required", http.StatusBadRequest)
		return
	}

	var req UpdateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
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
	validSeverities := map[string]bool{"LOW": true, "MEDIUM": true, "HIGH": true, "CRITICAL": true, "*": true}
	if !validSeverities[req.Severity] {
		http.Error(w, "severity must be one of: LOW, MEDIUM, HIGH, CRITICAL, or * (wildcard)", http.StatusBadRequest)
		return
	}

	// Validate that not all fields are wildcards
	if req.Severity == "*" && req.Source == "*" && req.Name == "*" {
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

	// Publish rule.changed event after successful DB commit
	changed := &events.RuleChanged{
		RuleID:        rule.RuleID,
		ClientID:      rule.ClientID,
		Action:        events.ActionUpdated,
		Version:       rule.Version,
		UpdatedAt:     rule.UpdatedAt.Unix(),
		SchemaVersion: SchemaVersion,
	}
	if err := h.producer.Publish(ctx, changed); err != nil {
		slog.Error("Failed to publish rule.changed event", "error", err, "rule_id", rule.RuleID)
		// Continue - the rule was updated, event publishing failure can be handled separately
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rule)
}

// ToggleRuleEnabledRequest represents a request to toggle rule enabled status.
type ToggleRuleEnabledRequest struct {
	Enabled bool `json:"enabled"`
	Version int  `json:"version"` // Optimistic locking version
}

// ToggleRuleEnabled toggles the enabled status of a rule and publishes a rule.changed event.
func (h *Handlers) ToggleRuleEnabled(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ruleID := r.URL.Query().Get("rule_id")
	if ruleID == "" {
		http.Error(w, "rule_id query parameter is required", http.StatusBadRequest)
		return
	}

	var req ToggleRuleEnabledRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
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

	// Publish rule.changed event after successful DB commit
	action := events.ActionDisabled
	if rule.Enabled {
		action = events.ActionUpdated // Re-enabling is treated as update
	}
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
		// Continue - the rule was updated, event publishing failure can be handled separately
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rule)
}

// DeleteRule deletes a rule and publishes a rule.changed event.
func (h *Handlers) DeleteRule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ruleID := r.URL.Query().Get("rule_id")
	if ruleID == "" {
		http.Error(w, "rule_id query parameter is required", http.StatusBadRequest)
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

// ============================================================================
// Endpoint Handlers
// ============================================================================

// CreateEndpointRequest represents a request to create an endpoint.
type CreateEndpointRequest struct {
	RuleID string `json:"rule_id"`
	Type   string `json:"type"`   // email, webhook, slack
	Value  string `json:"value"`  // email address, URL, etc.
}

// CreateEndpoint creates a new endpoint for a rule.
func (h *Handlers) CreateEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateEndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
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
	validTypes := map[string]bool{"email": true, "webhook": true, "slack": true}
	if !validTypes[req.Type] {
		http.Error(w, "type must be one of: email, webhook, slack", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	endpoint, err := h.db.CreateEndpoint(ctx, req.RuleID, req.Type, req.Value)
	if err != nil {
		slog.Error("Failed to create endpoint", "error", err, "rule_id", req.RuleID)
		if strings.Contains(err.Error(), "rule not found") {
			http.Error(w, "Rule not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to create endpoint: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(endpoint)
}

// GetEndpoint retrieves an endpoint by ID.
func (h *Handlers) GetEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	endpointID := r.URL.Query().Get("endpoint_id")
	if endpointID == "" {
		http.Error(w, "endpoint_id query parameter is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	endpoint, err := h.db.GetEndpoint(ctx, endpointID)
	if err != nil {
		slog.Error("Failed to get endpoint", "error", err, "endpoint_id", endpointID)
		http.Error(w, "Endpoint not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(endpoint)
}

// ListEndpoints retrieves all endpoints for a rule.
func (h *Handlers) ListEndpoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ruleID := r.URL.Query().Get("rule_id")
	if ruleID == "" {
		http.Error(w, "rule_id query parameter is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	endpoints, err := h.db.ListEndpoints(ctx, ruleID)
	if err != nil {
		slog.Error("Failed to list endpoints", "error", err, "rule_id", ruleID)
		http.Error(w, "Failed to list endpoints", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(endpoints)
}

// UpdateEndpointRequest represents a request to update an endpoint.
type UpdateEndpointRequest struct {
	Type  string `json:"type"`  // email, webhook, slack
	Value string `json:"value"` // email address, URL, etc.
}

// UpdateEndpoint updates an endpoint.
func (h *Handlers) UpdateEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	endpointID := r.URL.Query().Get("endpoint_id")
	if endpointID == "" {
		http.Error(w, "endpoint_id query parameter is required", http.StatusBadRequest)
		return
	}

	var req UpdateEndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
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
	validTypes := map[string]bool{"email": true, "webhook": true, "slack": true}
	if !validTypes[req.Type] {
		http.Error(w, "type must be one of: email, webhook, slack", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	endpoint, err := h.db.UpdateEndpoint(ctx, endpointID, req.Type, req.Value)
	if err != nil {
		slog.Error("Failed to update endpoint", "error", err, "endpoint_id", endpointID)
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Endpoint not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to update endpoint: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(endpoint)
}

// ToggleEndpointEnabledRequest represents a request to toggle endpoint enabled status.
type ToggleEndpointEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

// ToggleEndpointEnabled toggles the enabled status of an endpoint.
func (h *Handlers) ToggleEndpointEnabled(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	endpointID := r.URL.Query().Get("endpoint_id")
	if endpointID == "" {
		http.Error(w, "endpoint_id query parameter is required", http.StatusBadRequest)
		return
	}

	var req ToggleEndpointEnabledRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	endpoint, err := h.db.ToggleEndpointEnabled(ctx, endpointID, req.Enabled)
	if err != nil {
		slog.Error("Failed to toggle endpoint enabled", "error", err, "endpoint_id", endpointID)
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Endpoint not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to toggle endpoint enabled: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(endpoint)
}

// DeleteEndpoint deletes an endpoint.
func (h *Handlers) DeleteEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	endpointID := r.URL.Query().Get("endpoint_id")
	if endpointID == "" {
		http.Error(w, "endpoint_id query parameter is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if err := h.db.DeleteEndpoint(ctx, endpointID); err != nil {
		slog.Error("Failed to delete endpoint", "error", err, "endpoint_id", endpointID)
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Endpoint not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to delete endpoint", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Notification Handlers
// ============================================================================

// GetNotification retrieves a notification by ID.
func (h *Handlers) GetNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	notificationID := r.URL.Query().Get("notification_id")
	if notificationID == "" {
		http.Error(w, "notification_id query parameter is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	notification, err := h.db.GetNotification(ctx, notificationID)
	if err != nil {
		slog.Error("Failed to get notification", "error", err, "notification_id", notificationID)
		http.Error(w, "Notification not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notification)
}

// ListNotifications retrieves all notifications, optionally filtered by client_id or status.
func (h *Handlers) ListNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID := r.URL.Query().Get("client_id")
	status := r.URL.Query().Get("status")

	var clientIDPtr *string
	if clientID != "" {
		clientIDPtr = &clientID
	}

	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}

	ctx := r.Context()
	notifications, err := h.db.ListNotifications(ctx, clientIDPtr, statusPtr)
	if err != nil {
		slog.Error("Failed to list notifications", "error", err)
		http.Error(w, "Failed to list notifications", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}
