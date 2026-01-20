// Package api provides HTTP API handlers and job management for alert-producer.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// HandleGenerate handles POST /api/v1/alerts/generate
func HandleGenerate(jm *JobManager, defaultKafkaBrokers string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		var req GenerateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
			return
		}

		// Validate configuration before creating job
		cfg, err := req.ToConfig(defaultKafkaBrokers)
		if err != nil {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid configuration: %v", err))
			return
		}
		
		// For single_test mode, skip distribution validation (not needed for custom single alert)
		// For other modes, validate everything including distributions
		if err := validateConfig(&cfg, req.SingleTest); err != nil {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("Configuration validation failed: %v", err))
			return
		}
		
		// Validate single alert properties if single_test is enabled
		if req.SingleTest {
			if req.Severity == "" && req.Source == "" && req.Name == "" {
				// All empty - use defaults, this is fine
			} else {
				// At least one is provided, validate severity if provided
				if req.Severity != "" {
					validSeverities := map[string]bool{"LOW": true, "MEDIUM": true, "HIGH": true, "CRITICAL": true}
					if !validSeverities[req.Severity] {
						respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid severity: %s (must be LOW, MEDIUM, HIGH, or CRITICAL)", req.Severity))
						return
					}
				}
			}
		}

		// Create job
		job := jm.CreateJob(&req)

		// Start job
		jm.RunJob(job, defaultKafkaBrokers)

		respondJSON(w, http.StatusAccepted, GenerateResponse{
			JobID:  job.ID,
			Status: string(job.GetStatus()),
		})
	}
}
