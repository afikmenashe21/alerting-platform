// Package processor provides alert processing orchestration with support for
// different execution modes (burst, continuous, test).
package processor

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// runBurstMode sends a fixed number of alerts immediately without rate limiting.
func (p *Processor) runBurstMode(ctx context.Context) error {
	return p.runBurstModeWithSize(ctx, p.cfg.BurstSize, nil)
}

// runBurstModeWithSize sends a fixed number of alerts immediately.
// If progressCallback is provided, it will be called after each alert is sent.
func (p *Processor) runBurstModeWithSize(ctx context.Context, burstSize int, progressCallback func(sent int)) error {
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

		alertStart := time.Now()
		alert := p.generator.Generate()
		if err := p.publisher.Publish(ctx, alert); err != nil {
			p.metrics.RecordError()
			if err := handlePublishError(ctx, alert, err, i+1); err == context.Canceled {
				slog.Warn("Publish cancelled during burst", "sent", i, "requested", burstSize)
				return context.Canceled
			}
			return fmt.Errorf("failed to publish alert %d: %w", i+1, err)
		}

		p.metrics.RecordProcessed(time.Since(alertStart))
		p.metrics.RecordPublished()

		// Update progress callback if provided
		if progressCallback != nil {
			progressCallback(i + 1)
		}

		// Log first alert with full details for verification
		if i == 0 {
			logAlertDetails("Published first alert (sample)", alert)
		}

		// Log progress periodically to avoid log spam
		if (i+1)%burstProgressInterval == 0 {
			elapsed := time.Since(startTime)
			rate := calculateRate(i+1, elapsed)
			slog.Info("Burst progress",
				"sent", i+1,
				"total", burstSize,
				"rate_per_sec", formatRate(rate),
			)
		}
	}

	elapsed := time.Since(startTime)
	rate := calculateRate(burstSize, elapsed)
	slog.Info("Burst mode completed",
		"total_sent", burstSize,
		"duration_sec", formatDuration(elapsed),
		"rate_per_sec", formatRate(rate),
	)
	return nil
}
