// Package processor provides alert evaluation processing orchestration.
// It handles consuming alerts, matching against rules, and publishing matched alerts.
package processor

import (
	"context"
	"log/slog"

	"evaluator/internal/consumer"
	"evaluator/internal/matcher"
	"evaluator/internal/producer"

	"github.com/afikmenashe/alerting-platform/pkg/metrics"
)

// Processor orchestrates alert evaluation and matching.
type Processor struct {
	consumer *consumer.Consumer
	producer *producer.Producer
	matcher  *matcher.Matcher
	metrics  Metrics
	// rawMetrics holds the original collector for external access via GetMetrics().
	rawMetrics *metrics.Collector
}

// NewProcessor creates a new alert evaluation processor without metrics.
func NewProcessor(consumer *consumer.Consumer, producer *producer.Producer, matcher *matcher.Matcher) *Processor {
	return &Processor{
		consumer:   consumer,
		producer:   producer,
		matcher:    matcher,
		metrics:    NoOpMetrics{},
		rawMetrics: nil,
	}
}

// NewProcessorWithMetrics creates a processor with a shared metrics collector.
func NewProcessorWithMetrics(consumer *consumer.Consumer, producer *producer.Producer, matcher *matcher.Matcher, m *metrics.Collector) *Processor {
	return &Processor{
		consumer:   consumer,
		producer:   producer,
		matcher:    matcher,
		metrics:    wrapMetrics(m),
		rawMetrics: m,
	}
}

// ProcessAlerts continuously reads alerts from Kafka, matches them against rules,
// and publishes matched alerts to the output topic.
//
// Commit policy: offsets are committed only when all publishes for an alert succeed.
// This ensures at-least-once delivery semantics.
func (p *Processor) ProcessAlerts(ctx context.Context) error {
	slog.Info("Starting alert processing loop")

	// Record initial rule count
	p.metrics.AddCustom("rules_count", uint64(p.matcher.RuleCount()))

	for {
		if err := p.processNextMessage(ctx); err != nil {
			return err
		}
	}
}

// processNextMessage reads and processes a single message from Kafka.
// Returns nil to continue the loop, or an error to stop (context cancellation).
func (p *Processor) processNextMessage(ctx context.Context) error {
	select {
	case <-ctx.Done():
		slog.Info("Alert processing loop stopped")
		return ctx.Err()
	default:
	}

	// Read alert from Kafka
	alert, msg, err := p.consumer.ReadMessage(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return nil // Context cancelled, exit gracefully
		}
		slog.Error("Failed to read alert", "error", err)
		return nil // Continue processing
	}

	p.metrics.RecordReceived()

	// Process the alert (match + publish)
	result := p.processOne(ctx, alert)

	// Commit offset only after all publishes succeeded
	if result.allPublishesSucceeded {
		if err := p.consumer.CommitMessage(ctx, msg); err != nil {
			slog.Error("Failed to commit offset",
				"alert_id", alert.AlertID,
				"error", err,
			)
			p.metrics.RecordError()
		}
	} else {
		slog.Warn("Skipping offset commit due to publish failures, message will be redelivered",
			"alert_id", alert.AlertID,
		)
	}

	return nil
}

// GetMetrics returns the underlying metrics collector for external access.
// Returns nil if the processor was created without metrics.
func (p *Processor) GetMetrics() *metrics.Collector {
	return p.rawMetrics
}
