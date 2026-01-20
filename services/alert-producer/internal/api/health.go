// Package api provides HTTP API handlers and job management for alert-producer.
package api

import (
	"net/http"
)

// HandleHealth handles GET /health
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
