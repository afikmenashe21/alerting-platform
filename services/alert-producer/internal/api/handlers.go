package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"alert-producer/internal/config"
)

// GenerateRequest represents a request to generate alerts.
type GenerateRequest struct {
	RPS          *float64 `json:"rps,omitempty"`
	Duration     string   `json:"duration,omitempty"` // e.g., "60s", "5m"
	BurstSize    *int     `json:"burst,omitempty"`
	Seed         *int64   `json:"seed,omitempty"`
	SeverityDist string   `json:"severity_dist,omitempty"`
	SourceDist   string   `json:"source_dist,omitempty"`
	NameDist     string   `json:"name_dist,omitempty"`
	KafkaBrokers string   `json:"kafka_brokers,omitempty"`
	Topic        string   `json:"topic,omitempty"`
	Mock         bool     `json:"mock,omitempty"`
	Test         bool     `json:"test,omitempty"`
	SingleTest   bool     `json:"single_test,omitempty"`
	// Single alert properties (used when single_test is true)
	Severity     string   `json:"severity,omitempty"` // e.g., "HIGH", "LOW", "MEDIUM", "CRITICAL"
	Source       string   `json:"source,omitempty"`   // e.g., "api", "db", "cache"
	Name         string   `json:"name,omitempty"`     // e.g., "timeout", "error", "crash"
}

// ToConfig converts a GenerateRequest to a config.Config.
func (req *GenerateRequest) ToConfig(defaultKafkaBrokers string) (config.Config, error) {
	cfg := config.Config{
		KafkaBrokers: defaultKafkaBrokers,
		Topic:        "alerts.new",
		RPS:          10.0,
		Duration:     60 * time.Second,
		BurstSize:    0,
		Seed:         0,
		SeverityDist: "HIGH:30,MEDIUM:30,LOW:25,CRITICAL:15",
		SourceDist:   "api:25,db:20,cache:15,monitor:15,queue:10,worker:5,frontend:5,backend:5",
		NameDist:     "timeout:15,error:15,crash:10,slow:10,memory:10,cpu:10,disk:10,network:10,auth:5,validation:5",
	}

	// Override with request values
	if req.KafkaBrokers != "" {
		cfg.KafkaBrokers = req.KafkaBrokers
	}
	if req.Topic != "" {
		cfg.Topic = req.Topic
	}
	if req.RPS != nil {
		cfg.RPS = *req.RPS
	}
	if req.Duration != "" {
		duration, err := time.ParseDuration(req.Duration)
		if err != nil {
			return cfg, fmt.Errorf("invalid duration format: %w", err)
		}
		cfg.Duration = duration
	}
	if req.BurstSize != nil {
		cfg.BurstSize = *req.BurstSize
	}
	if req.Seed != nil {
		cfg.Seed = *req.Seed
	}
	if req.SeverityDist != "" {
		cfg.SeverityDist = req.SeverityDist
	}
	if req.SourceDist != "" {
		cfg.SourceDist = req.SourceDist
	}
	if req.NameDist != "" {
		cfg.NameDist = req.NameDist
	}

	return cfg, nil
}

// GenerateResponse represents the response to a generate request.
type GenerateResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

// JobResponse represents a job status response.
type JobResponse struct {
	ID          string    `json:"id"`
	Status      string    `json:"status"`
	Config      *GenerateRequest `json:"config"`
	CreatedAt   time.Time `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	AlertsSent  int64     `json:"alerts_sent"`
	Error       string    `json:"error,omitempty"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

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

// HandleGetJob handles GET /api/v1/alerts/generate/:jobId
func HandleGetJob(jm *JobManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		jobID := r.URL.Query().Get("job_id")
		if jobID == "" {
			respondError(w, http.StatusBadRequest, "job_id parameter is required")
			return
		}

		job, ok := jm.GetJob(jobID)
		if !ok {
			respondError(w, http.StatusNotFound, "Job not found")
			return
		}

		respondJSON(w, http.StatusOK, jobToResponse(job))
	}
}

// HandleListJobs handles GET /api/v1/alerts/generate
func HandleListJobs(jm *JobManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		statusFilter := JobStatus(r.URL.Query().Get("status"))
		jobs := jm.ListJobs(statusFilter)

		responses := make([]JobResponse, len(jobs))
		for i, job := range jobs {
			responses[i] = jobToResponse(job)
		}

		respondJSON(w, http.StatusOK, responses)
	}
}

// HandleStopJob handles POST /api/v1/alerts/generate/:jobId/stop
func HandleStopJob(jm *JobManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		jobID := r.URL.Query().Get("job_id")
		if jobID == "" {
			respondError(w, http.StatusBadRequest, "job_id parameter is required")
			return
		}

		job, ok := jm.GetJob(jobID)
		if !ok {
			respondError(w, http.StatusNotFound, "Job not found")
			return
		}

		// Check if job can be cancelled
		currentStatus := job.GetStatus()
		if currentStatus != JobStatusPending && currentStatus != JobStatusRunning {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("Job cannot be cancelled. Current status: %s", currentStatus))
			return
		}

		// Cancel the job (this cancels the context, goroutine will update status)
		job.Cancel()

		// Wait a moment for the goroutine to detect cancellation and update status
		time.Sleep(100 * time.Millisecond)

		// Get updated job status
		updatedJob, _ := jm.GetJob(jobID)
		respondJSON(w, http.StatusOK, jobToResponse(updatedJob))
	}
}

// HandleHealth handles GET /health
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

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

// parseInt parses an integer from a string, returning 0 if empty or invalid.
func parseInt(s string) int {
	if s == "" {
		return 0
	}
	val, _ := strconv.Atoi(s)
	return val
}

// parseFloat64 parses a float64 from a string, returning 0 if empty or invalid.
func parseFloat64(s string) float64 {
	if s == "" {
		return 0
	}
	val, _ := strconv.ParseFloat(s, 64)
	return val
}
