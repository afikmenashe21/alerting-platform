// Package processor provides alert evaluation processing orchestration.
// It handles consuming alerts, matching against rules, and publishing matched alerts.
package processor

import (
	"context"
	"log/slog"
	"time"

	"evaluator/internal/consumer"
	"evaluator/internal/events"
	"evaluator/internal/matcher"
	"evaluator/internal/producer"

	"github.com/afikmenashe/alerting-platform/pkg/metrics"
)

// Processor orchestrates alert evaluation and matching.
type Processor struct {
	consumer *consumer.Consumer
	producer *producer.Producer
	matcher  *matcher.Matcher
	metrics  *metrics.Collector
}

// NewProcessor creates a new alert evaluation processor (without shared metrics).
func NewProcessor(consumer *consumer.Consumer, producer *producer.Producer, matcher *matcher.Matcher) *Processor {
	return &Processor{
		consumer: consumer,
		producer: producer,
		matcher:  matcher,
		metrics:  nil, // No metrics collector
	}
}

// NewProcessorWithMetrics creates a processor with shared metrics collector.
func NewProcessorWithMetrics(consumer *consumer.Consumer, producer *producer.Producer, matcher *matcher.Matcher, m *metrics.Collector) *Processor {
	return &Processor{
		consumer: consumer,
		producer: producer,
		matcher:  matcher,
		metrics:  m,
	}
}

// ProcessAlerts continuously reads alerts from Kafka, matches them against rules,
// and publishes matched alerts to the output topic.
func (p *Processor) ProcessAlerts(ctx context.Context) error {
	slog.Info("Starting alert processing loop")

	// Update rules count as custom counter
	if p.metrics != nil {
		p.metrics.AddCustom("rules_count", uint64(p.matcher.RuleCount()))
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("Alert processing loop stopped")
			return nil
		default:
			// Start timing for full processing latency
			startTime := time.Now()

			// Read alert from Kafka
			alert, msg, err := p.consumer.ReadMessage(ctx)
			if err != nil {
				// Check if context was cancelled
				if ctx.Err() != nil {
					return nil
				}
				slog.Error("Failed to read alert", "error", err)
				continue
			}

			if p.metrics != nil {
				p.metrics.RecordReceived()
			}

			// Match alert against rules
			matches := p.matcher.Match(alert.Severity, alert.Source, alert.Name)

			// Track if all publishes succeeded for commit decision
			allPublishesSucceeded := true
			publishedCount := 0

			// Publish one message per client_id
			if len(matches) > 0 {
				for clientID, ruleIDs := range matches {
					matched := events.NewAlertMatched(alert, clientID, ruleIDs)

					if err := p.producer.Publish(ctx, matched); err != nil {
						slog.Error("Failed to publish matched alert",
							"alert_id", alert.AlertID,
							"client_id", clientID,
							"error", err,
						)
						if p.metrics != nil {
							p.metrics.RecordError()
						}
						allPublishesSucceeded = false
						continue
					}
					publishedCount++
					if p.metrics != nil {
						p.metrics.RecordPublished()
					}

					slog.Debug("Published matched alert",
						"alert_id", alert.AlertID,
						"client_id", clientID,
						"rule_ids", ruleIDs,
					)
				}
				if p.metrics != nil {
					p.metrics.RecordProcessed(time.Since(startTime))
					p.metrics.IncrementCustom("alerts_matched")
				}
			} else {
				if p.metrics != nil {
					p.metrics.RecordProcessed(time.Since(startTime))
					p.metrics.IncrementCustom("alerts_unmatched")
				}
			}

			// Commit offset only after all publishes succeeded (or no matches)
			if allPublishesSucceeded {
				if err := p.consumer.CommitMessage(ctx, msg); err != nil {
					slog.Error("Failed to commit offset",
						"alert_id", alert.AlertID,
						"error", err,
					)
					if p.metrics != nil {
						p.metrics.RecordError()
					}
				}
			} else {
				slog.Warn("Skipping offset commit due to publish failures, message will be redelivered",
					"alert_id", alert.AlertID,
				)
			}
		}
	}
}

// GetMetrics returns the metrics collector for external access.
func (p *Processor) GetMetrics() *metrics.Collector {
	return p.metrics
}
