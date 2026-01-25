package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// Keep validation logic centralized to avoid divergence across endpoints.

var validSeverities = map[string]struct{}{
	"LOW":      {},
	"MEDIUM":   {},
	"HIGH":     {},
	"CRITICAL": {},
	"*":        {}, // wildcard
}

func isValidSeverity(severity string) bool {
	_, ok := validSeverities[severity]
	return ok
}

func isAllWildcards(severity, source, name string) bool {
	return severity == "*" && source == "*" && name == "*"
}

var validEndpointTypes = map[string]struct{}{
	"email":   {},
	"webhook": {},
	"slack":   {},
}

func isValidEndpointType(t string) bool {
	_, ok := validEndpointTypes[t]
	return ok
}

// HTTP helper functions to reduce duplication across handlers.

// requireMethod validates that the request method matches the expected method.
// Returns true if valid, false otherwise (and writes error response).
func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

// decodeJSON decodes the request body as JSON into the provided value.
// Returns true on success, false on error (and writes error response).
func decodeJSON(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return false
	}
	return true
}

// writeJSON writes the value as JSON with appropriate headers.
func writeJSON(w http.ResponseWriter, statusCode int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(v)
}

// requireQueryParam extracts a query parameter and validates it's not empty.
// Returns the value and true if valid, empty string and false otherwise (and writes error response).
func requireQueryParam(w http.ResponseWriter, r *http.Request, paramName string) (string, bool) {
	value := r.URL.Query().Get(paramName)
	if value == "" {
		http.Error(w, paramName+" query parameter is required", http.StatusBadRequest)
		return "", false
	}
	return value, true
}

// Pagination holds parsed pagination parameters.
type Pagination struct {
	Limit  int
	Offset int
}

// DefaultPagination contains the default pagination values.
var DefaultPagination = Pagination{Limit: 50, Offset: 0}

// parsePagination extracts limit and offset from query parameters.
// Uses defaults if not provided or invalid.
func parsePagination(r *http.Request) Pagination {
	p := DefaultPagination

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			p.Limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			p.Offset = o
		}
	}

	return p
}
