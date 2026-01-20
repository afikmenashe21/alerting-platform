// Package handlers provides HTTP handlers for the rule-service API.
package handlers

import (
	"log/slog"
	"net/http"
	"strings"
)

// handleDBError handles database errors and writes appropriate HTTP responses.
// Returns true if error was handled, false otherwise.
func handleDBError(w http.ResponseWriter, err error, resource string, resourceID string) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	slog.Error("Database error", "error", err, "resource", resource, "resource_id", resourceID)

	// Handle specific error cases
	if strings.Contains(errStr, "not found") {
		http.Error(w, strings.Title(resource)+" not found", http.StatusNotFound)
		return true
	}
	if strings.Contains(errStr, "version mismatch") {
		http.Error(w, errStr, http.StatusConflict)
		return true
	}
	if strings.Contains(errStr, "already exists") {
		http.Error(w, strings.Title(resource)+" already exists", http.StatusConflict)
		return true
	}
	if strings.Contains(errStr, "client not found") {
		http.Error(w, "Client not found", http.StatusNotFound)
		return true
	}

	// Generic error
	http.Error(w, "Failed to "+strings.ToLower(resource)+": "+errStr, http.StatusBadRequest)
	return true
}

// validateRuleFields validates rule fields (severity, source, name) are not empty.
// Returns true if valid, false otherwise (and writes error response).
func validateRuleFields(w http.ResponseWriter, severity, source, name string) bool {
	if severity == "" {
		http.Error(w, "severity is required", http.StatusBadRequest)
		return false
	}
	if source == "" {
		http.Error(w, "source is required", http.StatusBadRequest)
		return false
	}
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return false
	}
	return true
}

// validateRuleValues validates rule values (severity enum and wildcard rules).
// Returns true if valid, false otherwise (and writes error response).
func validateRuleValues(w http.ResponseWriter, severity, source, name string) bool {
	// Validate severity enum (allow "*" as wildcard)
	if !isValidSeverity(severity) {
		http.Error(w, "severity must be one of: LOW, MEDIUM, HIGH, CRITICAL, or * (wildcard)", http.StatusBadRequest)
		return false
	}

	// Validate that not all fields are wildcards
	if isAllWildcards(severity, source, name) {
		http.Error(w, "cannot create rule with all fields as wildcards (*)", http.StatusBadRequest)
		return false
	}

	return true
}
