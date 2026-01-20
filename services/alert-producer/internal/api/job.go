// Package api provides HTTP API handlers and job management for alert-producer.
package api

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// JobStatus represents the status of an alert generation job.
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// Job represents a single alert generation job.
type Job struct {
	ID          string             `json:"id"`
	Status      JobStatus          `json:"status"`
	Config      *GenerateRequest   `json:"config"`
	CreatedAt   time.Time          `json:"created_at"`
	StartedAt   *time.Time         `json:"started_at,omitempty"`
	CompletedAt *time.Time         `json:"completed_at,omitempty"`
	AlertsSent  int64              `json:"alerts_sent"`
	Error       string             `json:"error,omitempty"`
	cancelFunc  context.CancelFunc `json:"-"`
	mu          sync.RWMutex       `json:"-"`
}

// JobManager manages alert generation jobs.
type JobManager struct {
	jobs map[string]*Job
	mu   sync.RWMutex
}

// NewJobManager creates a new job manager.
func NewJobManager() *JobManager {
	return &JobManager{
		jobs: make(map[string]*Job),
	}
}

// CreateJob creates a new job and returns its ID.
func (jm *JobManager) CreateJob(req *GenerateRequest) *Job {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	job := &Job{
		ID:        generateJobID(),
		Status:    JobStatusPending,
		Config:    req,
		CreatedAt: time.Now(),
	}

	jm.jobs[job.ID] = job
	return job
}

// GetJob retrieves a job by ID.
func (jm *JobManager) GetJob(id string) (*Job, bool) {
	jm.mu.RLock()
	defer jm.mu.RUnlock()
	job, ok := jm.jobs[id]
	return job, ok
}

// ListJobs returns all jobs, optionally filtered by status.
func (jm *JobManager) ListJobs(statusFilter JobStatus) []*Job {
	jm.mu.RLock()
	defer jm.mu.RUnlock()

	var jobs []*Job
	for _, job := range jm.jobs {
		if statusFilter == "" || job.GetStatus() == statusFilter {
			jobs = append(jobs, job)
		}
	}
	return jobs
}

// UpdateJobStatus updates a job's status.
func (j *Job) UpdateStatus(status JobStatus) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = status
	if status == JobStatusRunning && j.StartedAt == nil {
		now := time.Now()
		j.StartedAt = &now
	}
	if status == JobStatusCompleted || status == JobStatusFailed || status == JobStatusCancelled {
		now := time.Now()
		j.CompletedAt = &now
	}
}

// SetError sets the error message for a job.
func (j *Job) SetError(err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if err != nil {
		j.Error = err.Error()
	}
}

// IncrementAlertsSent increments the alerts sent counter.
func (j *Job) IncrementAlertsSent() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.AlertsSent++
}

// SetAlertsSent sets the alerts sent counter.
func (j *Job) SetAlertsSent(count int64) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.AlertsSent = count
}

// GetStatus returns the current job status.
func (j *Job) GetStatus() JobStatus {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.Status
}

// GetAlertsSent returns the number of alerts sent.
func (j *Job) GetAlertsSent() int64 {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.AlertsSent
}

// SetCancelFunc sets the cancel function for the job.
func (j *Job) SetCancelFunc(cancel context.CancelFunc) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.cancelFunc = cancel
}

// Cancel cancels the job if it's running.
// It calls the cancel function to signal cancellation, but doesn't update status immediately.
// The goroutine will detect cancellation and update status to cancelled.
func (j *Job) Cancel() {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.cancelFunc != nil {
		// Call cancel function to signal cancellation to the goroutine
		j.cancelFunc()
		// Clear the cancel function to prevent double cancellation
		j.cancelFunc = nil
	}
	// Don't update status here - let the goroutine handle it when it detects cancellation
}

// generateJobID generates a unique job ID using UUID.
func generateJobID() string {
	return uuid.New().String()
}
