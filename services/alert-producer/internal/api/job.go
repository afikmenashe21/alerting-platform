// Package api provides HTTP API handlers and job management for alert-producer.
package api

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"alert-producer/internal/config"
	"alert-producer/internal/generator"
	"alert-producer/internal/processor"
	"alert-producer/internal/producer"

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

// RunJob executes a job in a goroutine.
func (jm *JobManager) RunJob(job *Job, kafkaBrokers string) {
	ctx, cancel := context.WithCancel(context.Background())
	job.SetCancelFunc(cancel)

	go func() {
		defer cancel()

		// Convert request to config
		cfg, err := job.Config.ToConfig(kafkaBrokers)
		if err != nil {
			job.UpdateStatus(JobStatusFailed)
			job.SetError(err)
			return
		}

		// Validate config (skip distribution validation for single_test mode)
		// Note: This is a fallback validation - main validation happens in handlers
		// For single_test, distributions aren't needed since it uses hardcoded test alert
		if err := validateConfigForJob(&cfg, job.Config.SingleTest); err != nil {
			job.UpdateStatus(JobStatusFailed)
			job.SetError(err)
			return
		}

		// Initialize producer
		var alertPublisher producer.AlertPublisher
		if job.Config.Mock {
			alertPublisher = producer.NewMock(cfg.Topic)
			defer alertPublisher.Close()
		} else {
			kafkaProd, err2 := producer.New(cfg.KafkaBrokers, cfg.Topic)
			if err2 != nil {
				job.UpdateStatus(JobStatusFailed)
				job.SetError(err2)
				return
			}
			alertPublisher = kafkaProd
			defer alertPublisher.Close()
		}

		// Initialize generator
		gen := generator.New(cfg)

		// Initialize processor
		proc := processor.NewProcessor(gen, alertPublisher, &cfg)

		// Update status to running
		job.UpdateStatus(JobStatusRunning)

		// Run appropriate mode
		var runErr error
		if job.Config.SingleTest {
			// Single test mode - use user-provided values or defaults
			severity := job.Config.Severity
			if severity == "" {
				severity = "LOW" // Default
			}
			source := job.Config.Source
			if source == "" {
				source = "test-source" // Default
			}
			name := job.Config.Name
			if name == "" {
				name = "test-name" // Default
			}
			customAlert := generator.GenerateCustomAlert(severity, source, name)
			runErr = alertPublisher.Publish(ctx, customAlert)
			if runErr == nil {
				job.IncrementAlertsSent()
			}
		} else if job.Config.Test {
			// Test mode - track progress in real-time
			if cfg.BurstSize > 0 {
				runErr = proc.ProcessTestBurstWithProgress(ctx, cfg.BurstSize, func(sent int) {
					job.SetAlertsSent(int64(sent))
				})
			} else {
				runErr = proc.ProcessTestContinuousWithProgress(ctx, cfg.RPS, cfg.Duration, func(sent int) {
					job.SetAlertsSent(int64(sent))
				})
			}
		} else if cfg.BurstSize > 0 {
			// Burst mode - track progress in real-time
			runErr = proc.ProcessBurstWithProgress(ctx, cfg.BurstSize, func(sent int) {
				job.SetAlertsSent(int64(sent))
			})
		} else {
			// Continuous mode - track progress in real-time
			runErr = proc.ProcessContinuousWithProgress(ctx, cfg.RPS, cfg.Duration, func(sent int) {
				job.SetAlertsSent(int64(sent))
			})
		}

		if runErr != nil {
			// Check if error is due to context cancellation
			if errors.Is(runErr, context.Canceled) || ctx.Err() == context.Canceled {
				// Job was cancelled - update status
				job.UpdateStatus(JobStatusCancelled)
				slog.Info("Job cancelled by user", "job_id", job.ID, "alerts_sent", job.GetAlertsSent())
			} else {
				job.UpdateStatus(JobStatusFailed)
				job.SetError(runErr)
			}
			return
		}

		// Check if context was cancelled before marking as completed
		select {
		case <-ctx.Done():
			job.UpdateStatus(JobStatusCancelled)
			slog.Info("Job cancelled before completion", "job_id", job.ID, "alerts_sent", job.GetAlertsSent())
		default:
			job.UpdateStatus(JobStatusCompleted)
		}
	}()
}

// validateConfigForJob validates configuration for job execution.
// Skips distribution and RPS/duration validation when singleTest is true.
// This is a wrapper around validateConfig for consistency.
func validateConfigForJob(cfg *config.Config, singleTest bool) error {
	return validateConfig(cfg, singleTest)
}
