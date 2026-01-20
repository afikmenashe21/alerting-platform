// Package api provides HTTP API handlers and job management for alert-producer.
package api

import (
	"context"
	"errors"
	"log/slog"

	"alert-producer/internal/config"
	"alert-producer/internal/generator"
	"alert-producer/internal/processor"
	"alert-producer/internal/producer"
)

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
