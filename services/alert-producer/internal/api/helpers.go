// Package api provides HTTP API handlers and job management for alert-producer.
package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"alert-producer/internal/config"
)

// jobToResponse converts a Job to a JobResponse.
func jobToResponse(job *Job) JobResponse {
	job.mu.RLock()
	defer job.mu.RUnlock()

	return JobResponse{
		ID:          job.ID,
		Status:      string(job.Status),
		Config:      job.Config,
		CreatedAt:   job.CreatedAt,
		StartedAt:   job.StartedAt,
		CompletedAt: job.CompletedAt,
		AlertsSent:  job.AlertsSent,
		Error:       job.Error,
	}
}

// respondJSON sends a JSON response.
func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Failed to encode JSON response", "error", err)
	}
}

// respondError sends an error response.
func respondError(w http.ResponseWriter, statusCode int, message string) {
	respondJSON(w, statusCode, ErrorResponse{Error: message})
}

// validateConfig validates the configuration, optionally skipping distribution validation.
// When skipDistributions is true (single_test mode), also skips RPS/duration validation.
func validateConfig(cfg *config.Config, skipDistributions bool) error {
	if cfg.KafkaBrokers == "" {
		return fmt.Errorf("kafka-brokers cannot be empty")
	}
	if cfg.Topic == "" {
		return fmt.Errorf("topic cannot be empty")
	}
	
	// For single_test mode, RPS/duration/burst are not needed (just sends one alert)
	if !skipDistributions {
		if cfg.RPS <= 0 && cfg.BurstSize <= 0 {
			return fmt.Errorf("rps must be > 0 or burst must be > 0")
		}
		if cfg.BurstSize == 0 && cfg.Duration <= 0 {
			return fmt.Errorf("duration must be > 0 when not in burst mode")
		}
	}
	
	// Skip distribution validation for single_test mode (not needed)
	if !skipDistributions {
		// Validate distribution strings
		if _, err := config.ParseDistribution(cfg.SeverityDist); err != nil {
			return fmt.Errorf("invalid severity-dist: %w", err)
		}
		if _, err := config.ParseDistribution(cfg.SourceDist); err != nil {
			return fmt.Errorf("invalid source-dist: %w", err)
		}
		if _, err := config.ParseDistribution(cfg.NameDist); err != nil {
			return fmt.Errorf("invalid name-dist: %w", err)
		}
	}
	
	return nil
}
