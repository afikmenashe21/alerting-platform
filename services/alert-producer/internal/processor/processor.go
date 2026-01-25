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
	metrics   MetricsRecorder
}

// NewProcessor creates a new alert processor.
// If metrics is nil, a no-op implementation is used.
func NewProcessor(gen *generator.Generator, pub producer.AlertPublisher, cfg *config.Config, metrics MetricsRecorder) *Processor {
	if metrics == nil {
		metrics = NoOpMetrics{}
	}
	return &Processor{
		generator: gen,
		publisher: pub,
		cfg:       cfg,
		metrics:   metrics,
	}
}

// Process runs the appropriate processing mode based on configuration.
func (p *Processor) Process(ctx context.Context) error {
	// Send boilerplate alert first
	boilerplateAlert := generator.GenerateBoilerplate()
	if err := p.publisher.Publish(ctx, boilerplateAlert); err != nil {
		if err := handlePublishError(ctx, boilerplateAlert, err, 0); err == context.Canceled {
			return context.Canceled
		}
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
	return p.runBurstModeWithSize(ctx, burstSize, nil)
}

// ProcessBurstWithProgress runs burst mode with progress callback.
func (p *Processor) ProcessBurstWithProgress(ctx context.Context, burstSize int, progressCallback func(sent int)) error {
	return p.runBurstModeWithSize(ctx, burstSize, progressCallback)
}

// ProcessContinuous runs continuous mode: generates and publishes alerts at a fixed rate.
func (p *Processor) ProcessContinuous(ctx context.Context, rps float64, duration time.Duration) error {
	return p.runContinuousModeWithParams(ctx, rps, duration, nil)
}

// ProcessContinuousWithProgress runs continuous mode with progress callback.
func (p *Processor) ProcessContinuousWithProgress(ctx context.Context, rps float64, duration time.Duration, progressCallback func(sent int)) error {
	return p.runContinuousModeWithParams(ctx, rps, duration, progressCallback)
}

// ProcessTest runs test mode: generates varied alerts including one test alert.
func (p *Processor) ProcessTest(ctx context.Context, rps float64, duration time.Duration, burstSize int) error {
	if burstSize > 0 {
		return p.runTestBurstMode(ctx, burstSize, nil)
	}
	return p.runTestContinuousMode(ctx, rps, duration, nil)
}

// ProcessTestBurstWithProgress runs test burst mode with progress callback.
func (p *Processor) ProcessTestBurstWithProgress(ctx context.Context, burstSize int, progressCallback func(sent int)) error {
	return p.runTestBurstMode(ctx, burstSize, progressCallback)
}

// ProcessTestContinuousWithProgress runs test continuous mode with progress callback.
func (p *Processor) ProcessTestContinuousWithProgress(ctx context.Context, rps float64, duration time.Duration, progressCallback func(sent int)) error {
	return p.runTestContinuousMode(ctx, rps, duration, progressCallback)
}


