// Package api provides HTTP API handlers and job management for alert-producer.
package api

import (
	"context"
	"errors"
	"log/slog"
	"time"

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

		cfg, err := job.Config.ToConfig(kafkaBrokers)
		if err != nil {
			job.fail(err)
			return
		}

		if err := validateConfig(&cfg, job.Config.SingleTest); err != nil {
			job.fail(err)
			return
		}

		alertPublisher, err := createPublisher(job.Config.Mock, cfg)
		if err != nil {
			job.fail(err)
			return
		}
		defer alertPublisher.Close()

		job.UpdateStatus(JobStatusRunning)
		runErr := job.execute(ctx, alertPublisher, &cfg)
		job.finalize(ctx, runErr)
	}()
}

// createPublisher initializes the appropriate alert publisher.
func createPublisher(mock bool, cfg config.Config) (producer.AlertPublisher, error) {
	if mock {
		return producer.NewMock(cfg.Topic), nil
	}
	return producer.New(cfg.KafkaBrokers, cfg.Topic)
}

// execute runs the appropriate job mode.
func (j *Job) execute(ctx context.Context, pub producer.AlertPublisher, cfg *config.Config) error {
	// Single alert mode
	if j.Config.SingleTest {
		return j.sendCustomAlerts(ctx, pub, 1, 0)
	}

	// Custom alerts with count
	hasCustom := j.Config.Severity != "" || j.Config.Source != "" || j.Config.Name != ""
	if hasCustom && j.Config.Count != nil && *j.Config.Count > 0 {
		interval := 0
		if j.Config.IntervalMs != nil {
			interval = *j.Config.IntervalMs
		}
		return j.sendCustomAlerts(ctx, pub, *j.Config.Count, interval)
	}

	// Standard modes via processor (nil metrics uses no-op implementation)
	gen := generator.New(*cfg)
	proc := processor.NewProcessor(gen, pub, cfg, nil)
	progress := func(sent int) { j.SetAlertsSent(int64(sent)) }

	if j.Config.Test {
		if cfg.BurstSize > 0 {
			return proc.ProcessTestBurstWithProgress(ctx, cfg.BurstSize, progress)
		}
		return proc.ProcessTestContinuousWithProgress(ctx, cfg.RPS, cfg.Duration, progress)
	}

	if cfg.BurstSize > 0 {
		return proc.ProcessBurstWithProgress(ctx, cfg.BurstSize, progress)
	}
	return proc.ProcessContinuousWithProgress(ctx, cfg.RPS, cfg.Duration, progress)
}

// sendCustomAlerts sends a specified number of custom alerts.
func (j *Job) sendCustomAlerts(ctx context.Context, pub producer.AlertPublisher, count, intervalMs int) error {
	severity, source, name := j.Config.Severity, j.Config.Source, j.Config.Name
	if severity == "" {
		severity = "LOW"
	}
	if source == "" {
		source = "test-source"
	}
	if name == "" {
		name = "test-name"
	}

	for i := 0; i < count && ctx.Err() == nil; i++ {
		if err := pub.Publish(ctx, generator.GenerateCustomAlert(severity, source, name)); err != nil {
			return err
		}
		j.IncrementAlertsSent()

		if intervalMs > 0 && i < count-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(intervalMs) * time.Millisecond):
			}
		}
	}
	return nil
}

// fail marks the job as failed with the given error.
func (j *Job) fail(err error) {
	j.UpdateStatus(JobStatusFailed)
	j.SetError(err)
}

// finalize sets the final job status based on execution result.
func (j *Job) finalize(ctx context.Context, err error) {
	if err != nil {
		if errors.Is(err, context.Canceled) || ctx.Err() == context.Canceled {
			j.UpdateStatus(JobStatusCancelled)
			slog.Info("Job cancelled", "job_id", j.ID, "alerts_sent", j.GetAlertsSent())
		} else {
			j.fail(err)
		}
		return
	}

	select {
	case <-ctx.Done():
		j.UpdateStatus(JobStatusCancelled)
		slog.Info("Job cancelled", "job_id", j.ID, "alerts_sent", j.GetAlertsSent())
	default:
		j.UpdateStatus(JobStatusCompleted)
	}
}
