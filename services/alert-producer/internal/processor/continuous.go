// Package processor provides alert processing orchestration with support for
// different execution modes (burst, continuous, test).
package processor

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// runContinuousMode generates and publishes alerts at a fixed rate (RPS) for a specified duration.
func (p *Processor) runContinuousMode(ctx context.Context) error {
	return p.runContinuousModeWithParams(ctx, p.cfg.RPS, p.cfg.Duration, nil)
}

// runContinuousModeWithParams generates and publishes alerts at a fixed rate.
// If progressCallback is provided, it will be called after each alert is sent.
func (p *Processor) runContinuousModeWithParams(ctx context.Context, rps float64, duration time.Duration, progressCallback func(sent int)) error {
	slog.Info("Starting continuous mode",
		"target_rps", rps,
		"duration", duration,
	)

	// Calculate ticker interval to achieve target RPS
	interval := time.Duration(float64(time.Second) / rps)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	deadline := time.Now().Add(duration)
	startTime := time.Now()
	totalSent := 0
	lastLog := time.Now()
	firstAlertLogged := false

	for {
		select {
		case <-ctx.Done():
			slog.Warn("Continuous mode cancelled",
				"sent", totalSent,
				"duration_requested", duration,
			)
			return ctx.Err()
		case <-ticker.C:
			// Check if we've exceeded the duration
			if time.Now().After(deadline) {
				elapsed := time.Since(startTime)
				actualRPS := calculateRate(totalSent, elapsed)
				slog.Info("Duration reached",
					"total_sent", totalSent,
					"duration_sec", formatDuration(elapsed),
					"target_rps", rps,
					"actual_rps", formatRate(actualRPS),
				)
				return nil
			}

			// Generate and publish alert
			alertStart := time.Now()
			alert := p.generator.Generate()
			if err := p.publisher.Publish(ctx, alert); err != nil {
				if p.metrics != nil {
					p.metrics.RecordError()
				}
				if err := handlePublishError(ctx, alert, err, totalSent+1); err == context.Canceled {
					slog.Warn("Publish cancelled during continuous", "sent", totalSent)
					return context.Canceled
				}
				return fmt.Errorf("failed to publish alert: %w", err)
			}

			totalSent++

			if p.metrics != nil {
				p.metrics.RecordProcessed(time.Since(alertStart))
				p.metrics.RecordPublished()
			}

			// Update progress callback if provided
			if progressCallback != nil {
				progressCallback(totalSent)
			}

			// Log first alert with full details for verification
			if !firstAlertLogged {
				logAlertDetails("Published first alert (sample)", alert)
				firstAlertLogged = true
			}

			// Log progress periodically with actual RPS calculation
			if time.Since(lastLog) >= progressLogInterval {
				elapsed := time.Since(startTime)
				actualRPS := calculateRate(totalSent, elapsed)
				slog.Info("Progress update",
					"sent", totalSent,
					"target_rps", rps,
					"actual_rps", formatRate(actualRPS),
					"elapsed_sec", formatDuration(elapsed),
				)
				lastLog = time.Now()
			}
		}
	}
}
