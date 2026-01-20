// Package processor provides alert processing orchestration with support for
// different execution modes (burst, continuous, test).
package processor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"alert-producer/internal/generator"
)

// isCancelled checks if an error is due to context cancellation.
func isCancelled(ctx context.Context, err error) bool {
	return errors.Is(err, context.Canceled) || ctx.Err() == context.Canceled
}

// handlePublishError handles publish errors, checking for context cancellation.
// Returns the error if it should be propagated, or context.Canceled if cancelled.
func handlePublishError(ctx context.Context, alert *generator.Alert, err error, alertNumber int) error {
	if isCancelled(ctx, err) {
		return context.Canceled
	}
	
	fields := []interface{}{
		"alert_id", alert.AlertID,
		"severity", alert.Severity,
		"source", alert.Source,
		"name", alert.Name,
		"error", err,
	}
	if alertNumber > 0 {
		fields = append(fields, "alert_number", alertNumber)
	}
	
	slog.Error("Failed to publish alert", fields...)
	return fmt.Errorf("failed to publish alert: %w", err)
}

// logAlertDetails logs alert details in a structured format.
func logAlertDetails(message string, alert *generator.Alert) {
	slog.Info(message,
		"alert_id", alert.AlertID,
		"severity", alert.Severity,
		"source", alert.Source,
		"name", alert.Name,
		"event_ts", alert.EventTS,
	)
}

// logAlertDetailsWithType logs alert details with a type indicator.
func logAlertDetailsWithType(message string, alert *generator.Alert, alertType string) {
	slog.Info(message,
		"type", alertType,
		"alert_id", alert.AlertID,
		"severity", alert.Severity,
		"source", alert.Source,
		"name", alert.Name,
		"event_ts", alert.EventTS,
	)
}

// calculateRate calculates the rate (items per second) given count and elapsed time.
func calculateRate(count int, elapsed time.Duration) float64 {
	seconds := elapsed.Seconds()
	if seconds <= 0 {
		return 0
	}
	return float64(count) / seconds
}

// formatDuration formats a duration as a string with 2 decimal places.
func formatDuration(d time.Duration) string {
	return fmt.Sprintf("%.2f", d.Seconds())
}

// formatRate formats a rate as a string with 2 decimal places.
func formatRate(rate float64) string {
	return fmt.Sprintf("%.2f", rate)
}
