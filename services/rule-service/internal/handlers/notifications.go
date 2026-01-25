// Package handlers provides HTTP handlers for the rule-service API.
package handlers

import (
	"log/slog"
	"net/http"
)

// GetNotification retrieves a notification by ID.
func (h *Handlers) GetNotification(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	notificationID, ok := requireQueryParam(w, r, "notification_id")
	if !ok {
		return
	}

	ctx := r.Context()
	notification, err := h.db.GetNotification(ctx, notificationID)
	if err != nil {
		slog.Error("Failed to get notification", "error", err, "notification_id", notificationID)
		http.Error(w, "Notification not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, notification)
}

// ListNotifications retrieves notifications with pagination, optionally filtered by client_id or status.
// Query params: client_id, status, limit (default 50, max 200), offset (default 0)
func (h *Handlers) ListNotifications(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
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

	p := parsePagination(r)
	ctx := r.Context()
	result, err := h.db.ListNotifications(ctx, clientIDPtr, statusPtr, p.Limit, p.Offset)
	if err != nil {
		slog.Error("Failed to list notifications", "error", err)
		http.Error(w, "Failed to list notifications", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, result)
}
