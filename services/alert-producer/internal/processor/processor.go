// Package processor provides alert processing orchestration with support for
// different execution modes (burst, continuous, test).
// It coordinates between the generator and producer to publish alerts.
package processor

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"alert-producer/internal/config"
	"alert-producer/internal/generator"
	"alert-producer/internal/producer"
)

const (
	// progressLogInterval defines how often to log progress in continuous mode
	progressLogInterval = 5 * time.Second
	// burstProgressInterval defines how often to log progress in burst mode (every N alerts)
	burstProgressInterval = 100
)

// Processor orchestrates alert generation and publishing.
type Processor struct {
	generator *generator.Generator
	publisher producer.AlertPublisher
	cfg       *config.Config
}

// NewProcessor creates a new alert processor.
func NewProcessor(gen *generator.Generator, pub producer.AlertPublisher, cfg *config.Config) *Processor {
	return &Processor{
		generator: gen,
		publisher: pub,
		cfg:       cfg,
	}
}

// Process runs the appropriate processing mode based on configuration.
func (p *Processor) Process(ctx context.Context) error {
	// Send boilerplate alert first
	boilerplateAlert := generator.GenerateBoilerplate()
	if err := p.publisher.Publish(ctx, boilerplateAlert); err != nil {
		slog.Error("Failed to publish boilerplate alert",
			"alert_id", boilerplateAlert.AlertID,
			"severity", boilerplateAlert.Severity,
			"source", boilerplateAlert.Source,
			"name", boilerplateAlert.Name,
			"error", err,
		)
		return fmt.Errorf("failed to publish boilerplate alert: %w", err)
	}
	slog.Info("Published boilerplate alert",
		"alert_id", boilerplateAlert.AlertID,
		"severity", boilerplateAlert.Severity,
		"source", boilerplateAlert.Source,
		"name", boilerplateAlert.Name,
	)

	// Run appropriate mode
	if p.cfg.BurstSize > 0 {
		return p.runBurstMode(ctx)
	}
	return p.runContinuousMode(ctx)
}

// ProcessBurst runs burst mode: sends a fixed number of alerts immediately.
func (p *Processor) ProcessBurst(ctx context.Context, burstSize int) error {
	return p.runBurstModeWithSize(ctx, burstSize)
}

// ProcessContinuous runs continuous mode: generates and publishes alerts at a fixed rate.
func (p *Processor) ProcessContinuous(ctx context.Context, rps float64, duration time.Duration) error {
	return p.runContinuousModeWithParams(ctx, rps, duration)
}

// ProcessTest runs test mode: generates varied alerts including one test alert.
func (p *Processor) ProcessTest(ctx context.Context, rps float64, duration time.Duration, burstSize int) error {
	if burstSize > 0 {
		return p.runTestBurstMode(ctx, burstSize)
	}
	return p.runTestContinuousMode(ctx, rps, duration)
}

// runBurstMode sends a fixed number of alerts immediately without rate limiting.
func (p *Processor) runBurstMode(ctx context.Context) error {
	return p.runBurstModeWithSize(ctx, p.cfg.BurstSize)
}

// runBurstModeWithSize sends a fixed number of alerts immediately.
func (p *Processor) runBurstModeWithSize(ctx context.Context, burstSize int) error {
	slog.Info("Starting burst mode", "total_alerts", burstSize)

	startTime := time.Now()
	for i := 0; i < burstSize; i++ {
		// Check for cancellation before each alert
		select {
		case <-ctx.Done():
			slog.Warn("Burst mode cancelled", "sent", i, "requested", burstSize)
			return ctx.Err()
		default:
		}

		alert := p.generator.Generate()
		if err := p.publisher.Publish(ctx, alert); err != nil {
			slog.Error("Failed to publish alert",
				"alert_id", alert.AlertID,
				"severity", alert.Severity,
				"source", alert.Source,
				"name", alert.Name,
				"error", err,
				"alert_number", i+1,
			)
			return fmt.Errorf("failed to publish alert %d: %w", i+1, err)
		}

		// Log first alert with full details for verification
		if i == 0 {
			logAlertDetails("Published first alert (sample)", alert)
		}

		// Log progress periodically to avoid log spam
		if (i+1)%burstProgressInterval == 0 {
			elapsed := time.Since(startTime).Seconds()
			rate := float64(i+1) / elapsed
			slog.Info("Burst progress",
				"sent", i+1,
				"total", burstSize,
				"rate_per_sec", fmt.Sprintf("%.2f", rate),
			)
		}
	}

	elapsed := time.Since(startTime).Seconds()
	rate := float64(burstSize) / elapsed
	slog.Info("Burst mode completed",
		"total_sent", burstSize,
		"duration_sec", fmt.Sprintf("%.2f", elapsed),
		"rate_per_sec", fmt.Sprintf("%.2f", rate),
	)
	return nil
}

// runContinuousMode generates and publishes alerts at a fixed rate (RPS) for a specified duration.
func (p *Processor) runContinuousMode(ctx context.Context) error {
	return p.runContinuousModeWithParams(ctx, p.cfg.RPS, p.cfg.Duration)
}

// runContinuousModeWithParams generates and publishes alerts at a fixed rate.
func (p *Processor) runContinuousModeWithParams(ctx context.Context, rps float64, duration time.Duration) error {
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
				elapsed := time.Since(startTime).Seconds()
				actualRPS := float64(totalSent) / elapsed
				slog.Info("Duration reached",
					"total_sent", totalSent,
					"duration_sec", fmt.Sprintf("%.2f", elapsed),
					"target_rps", rps,
					"actual_rps", fmt.Sprintf("%.2f", actualRPS),
				)
				return nil
			}

			// Generate and publish alert
			alert := p.generator.Generate()
			if err := p.publisher.Publish(ctx, alert); err != nil {
				slog.Error("Failed to publish alert",
					"alert_id", alert.AlertID,
					"severity", alert.Severity,
					"source", alert.Source,
					"name", alert.Name,
					"error", err,
					"total_sent", totalSent,
				)
				return fmt.Errorf("failed to publish alert: %w", err)
			}

			totalSent++

			// Log first alert with full details for verification
			if !firstAlertLogged {
				logAlertDetails("Published first alert (sample)", alert)
				firstAlertLogged = true
			}

			// Log progress periodically with actual RPS calculation
			if time.Since(lastLog) >= progressLogInterval {
				elapsed := time.Since(startTime).Seconds()
				actualRPS := float64(totalSent) / elapsed
				slog.Info("Progress update",
					"sent", totalSent,
					"target_rps", rps,
					"actual_rps", fmt.Sprintf("%.2f", actualRPS),
					"elapsed_sec", fmt.Sprintf("%.2f", elapsed),
				)
				lastLog = time.Now()
			}
		}
	}
}

// runTestBurstMode sends N varied alerts, including one test alert.
func (p *Processor) runTestBurstMode(ctx context.Context, burstSize int) error {
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
			slog.Error("Failed to publish alert",
				"alert_id", alert.AlertID,
				"severity", alert.Severity,
				"source", alert.Source,
				"name", alert.Name,
				"error", err,
				"alert_number", i+1,
			)
			return fmt.Errorf("failed to publish alert %d: %w", i+1, err)
		}

		if (i+1)%burstProgressInterval == 0 {
			elapsed := time.Since(startTime).Seconds()
			rate := float64(i+1) / elapsed
			slog.Info("Test mode burst progress",
				"sent", i+1,
				"total", burstSize,
				"rate_per_sec", fmt.Sprintf("%.2f", rate),
			)
		}
	}

	elapsed := time.Since(startTime).Seconds()
	rate := float64(burstSize) / elapsed
	slog.Info("Test mode burst completed",
		"total_sent", burstSize,
		"duration_sec", fmt.Sprintf("%.2f", elapsed),
		"rate_per_sec", fmt.Sprintf("%.2f", rate),
	)
	return nil
}

// runTestContinuousMode sends varied alerts at specified RPS, including one test alert.
func (p *Processor) runTestContinuousMode(ctx context.Context, rps float64, duration time.Duration) error {
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
				elapsed := time.Since(startTime).Seconds()
				actualRPS := float64(totalSent) / elapsed
				slog.Info("Test mode duration reached",
					"total_sent", totalSent,
					"duration_sec", fmt.Sprintf("%.2f", elapsed),
					"target_rps", rps,
					"actual_rps", fmt.Sprintf("%.2f", actualRPS),
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
				slog.Error("Failed to publish alert",
					"alert_id", alert.AlertID,
					"severity", alert.Severity,
					"source", alert.Source,
					"name", alert.Name,
					"error", err,
					"total_sent", totalSent,
				)
				return fmt.Errorf("failed to publish alert: %w", err)
			}

			totalSent++

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
				elapsed := time.Since(startTime).Seconds()
				actualRPS := float64(totalSent) / elapsed
				slog.Info("Test mode progress",
					"sent", totalSent,
					"target_rps", rps,
					"actual_rps", fmt.Sprintf("%.2f", actualRPS),
					"elapsed_sec", fmt.Sprintf("%.2f", elapsed),
					"test_alert_sent", testAlertSent,
				)
				lastLog = time.Now()
			}
		}
	}
}

// logAlertDetails logs alert details in a structured format.
func logAlertDetails(message string, alert *generator.Alert) {
	slog.Info(message,
		"alert_id", alert.AlertID,
		"severity", alert.Severity,
		"source", alert.Source,
		"name", alert.Name,
		"event_ts", alert.EventTS,
	)
}

// logAlertDetailsWithType logs alert details with a type indicator.
func logAlertDetailsWithType(message string, alert *generator.Alert, alertType string) {
	slog.Info(message,
		"type", alertType,
		"alert_id", alert.AlertID,
		"severity", alert.Severity,
		"source", alert.Source,
		"name", alert.Name,
		"event_ts", alert.EventTS,
	)
}
