// Package processor provides alert processing orchestration with support for
// different execution modes (burst, continuous, test).
package processor

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"alert-producer/internal/generator"
)

// runTestBurstMode sends N varied alerts, including one test alert.
// If progressCallback is provided, it will be called after each alert is sent.
func (p *Processor) runTestBurstMode(ctx context.Context, burstSize int, progressCallback func(sent int)) error {
	slog.Info("Test mode burst", "total_alerts", burstSize)
	startTime := time.Now()
	for i := 0; i < burstSize; i++ {
		select {
		case <-ctx.Done():
			slog.Warn("Test mode burst cancelled", "sent", i, "requested", burstSize)
			return ctx.Err()
		default:
		}

		var alert *generator.Alert
		// Include test alert once (at the beginning)
		if i == 0 {
			alert = generator.GenerateTestAlert()
			logAlertDetails("Published test alert (LOW/test-source/test-name)", alert)
		} else {
			// Generate varied alerts
			alert = p.generator.Generate()
		}

		if err := p.publisher.Publish(ctx, alert); err != nil {
			if err := handlePublishError(ctx, alert, err, i+1); err == context.Canceled {
				slog.Warn("Publish cancelled during test burst", "sent", i, "requested", burstSize)
				return context.Canceled
			}
			return fmt.Errorf("failed to publish alert %d: %w", i+1, err)
		}

		// Update progress callback if provided
		if progressCallback != nil {
			progressCallback(i + 1)
		}

		if (i+1)%burstProgressInterval == 0 {
			elapsed := time.Since(startTime)
			rate := calculateRate(i+1, elapsed)
			slog.Info("Test mode burst progress",
				"sent", i+1,
				"total", burstSize,
				"rate_per_sec", formatRate(rate),
			)
		}
	}

	elapsed := time.Since(startTime)
	rate := calculateRate(burstSize, elapsed)
	slog.Info("Test mode burst completed",
		"total_sent", burstSize,
		"duration_sec", formatDuration(elapsed),
		"rate_per_sec", formatRate(rate),
	)
	return nil
}

// runTestContinuousMode sends varied alerts at specified RPS, including one test alert.
// If progressCallback is provided, it will be called after each alert is sent.
func (p *Processor) runTestContinuousMode(ctx context.Context, rps float64, duration time.Duration, progressCallback func(sent int)) error {
	slog.Info("Test mode continuous",
		"target_rps", rps,
		"duration", duration,
	)

	interval := time.Duration(float64(time.Second) / rps)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	deadline := time.Now().Add(duration)
	startTime := time.Now()
	totalSent := 0
	lastLog := time.Now()
	firstAlertLogged := false
	testAlertSent := false

	for {
		select {
		case <-ctx.Done():
			slog.Warn("Test mode continuous cancelled",
				"sent", totalSent,
				"duration_requested", duration,
			)
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				elapsed := time.Since(startTime)
				actualRPS := calculateRate(totalSent, elapsed)
				slog.Info("Test mode duration reached",
					"total_sent", totalSent,
					"duration_sec", formatDuration(elapsed),
					"target_rps", rps,
					"actual_rps", formatRate(actualRPS),
					"test_alert_sent", testAlertSent,
				)
				return nil
			}

			var alert *generator.Alert
			// Include test alert once (at the beginning)
			if !testAlertSent {
				alert = generator.GenerateTestAlert()
				testAlertSent = true
			} else {
				// Generate varied alerts
				alert = p.generator.Generate()
			}

			if err := p.publisher.Publish(ctx, alert); err != nil {
				if err := handlePublishError(ctx, alert, err, totalSent+1); err == context.Canceled {
					slog.Warn("Publish cancelled during test continuous", "sent", totalSent)
					return context.Canceled
				}
				return fmt.Errorf("failed to publish alert: %w", err)
			}

			totalSent++

			// Update progress callback if provided
			if progressCallback != nil {
				progressCallback(totalSent)
			}

			if !firstAlertLogged {
				isTestAlert := alert.Severity == "LOW" && alert.Source == "test-source" && alert.Name == "test-name"
				alertType := "varied"
				if isTestAlert {
					alertType = "test"
				}
				logAlertDetailsWithType("Published first alert (sample)", alert, alertType)
				firstAlertLogged = true
			}

			if time.Since(lastLog) >= progressLogInterval {
				elapsed := time.Since(startTime)
				actualRPS := calculateRate(totalSent, elapsed)
				slog.Info("Test mode progress",
					"sent", totalSent,
					"target_rps", rps,
					"actual_rps", formatRate(actualRPS),
					"elapsed_sec", formatDuration(elapsed),
					"test_alert_sent", testAlertSent,
				)
				lastLog = time.Now()
			}
		}
	}
}
