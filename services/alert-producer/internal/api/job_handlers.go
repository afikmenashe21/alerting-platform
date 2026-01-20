// Package api provides HTTP API handlers and job management for alert-producer.
package api

import (
	"fmt"
	"net/http"
	"time"
)

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
