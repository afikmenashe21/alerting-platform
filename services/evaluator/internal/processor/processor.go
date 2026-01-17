// Package processor provides alert evaluation processing orchestration.
// It handles consuming alerts, matching against rules, and publishing matched alerts.
package processor

import (
	"context"
	"log/slog"

	"evaluator/internal/consumer"
	"evaluator/internal/events"
	"evaluator/internal/matcher"
	"evaluator/internal/producer"
)

// Processor orchestrates alert evaluation and matching.
type Processor struct {
	consumer *consumer.Consumer
	producer *producer.Producer
	matcher  *matcher.Matcher
}

// NewProcessor creates a new alert evaluation processor.
func NewProcessor(consumer *consumer.Consumer, producer *producer.Producer, matcher *matcher.Matcher) *Processor {
	return &Processor{
		consumer: consumer,
		producer: producer,
		matcher:  matcher,
	}
}

// ProcessAlerts continuously reads alerts from Kafka, matches them against rules,
// and publishes matched alerts to the output topic.
func (p *Processor) ProcessAlerts(ctx context.Context) error {
	slog.Info("Starting alert processing loop")

	for {
		select {
		case <-ctx.Done():
			slog.Info("Alert processing loop stopped")
			return nil
		default:
			// Read alert from Kafka
			alert, _, err := p.consumer.ReadMessage(ctx)
			if err != nil {
				// Check if context was cancelled
				if ctx.Err() != nil {
					return nil
				}
				slog.Error("Failed to read alert", "error", err)
				// Continue processing other messages
				continue
			}

			slog.Debug("Received alert",
				"alert_id", alert.AlertID,
				"severity", alert.Severity,
				"source", alert.Source,
				"name", alert.Name,
			)

			// Match alert against rules
			matches := p.matcher.Match(alert.Severity, alert.Source, alert.Name)

			// Publish one message per client_id
			if len(matches) > 0 {
				for clientID, ruleIDs := range matches {
					// Build matched alert event for this client
					matched := &events.AlertMatched{
						AlertID:       alert.AlertID,
						SchemaVersion: alert.SchemaVersion,
						EventTS:       alert.EventTS,
						Severity:      alert.Severity,
						Source:        alert.Source,
						Name:          alert.Name,
						Context:       alert.Context,
						ClientID:      clientID,
						RuleIDs:       ruleIDs,
					}

					// Publish message for this client
					if err := p.producer.Publish(ctx, matched); err != nil {
						slog.Error("Failed to publish matched alert",
							"alert_id", alert.AlertID,
							"client_id", clientID,
							"error", err,
						)
						// Continue processing other clients - Kafka will retry on next read
						continue
					}

					slog.Info("Published matched alert",
						"alert_id", alert.AlertID,
						"client_id", clientID,
						"rule_ids", ruleIDs,
					)
				}
			} else {
				slog.Info("No rules matched alert",
					"alert_id", alert.AlertID,
					"severity", alert.Severity,
					"source", alert.Source,
					"name", alert.Name,
				)
			}

			// Message is committed automatically by kafka-go after processing
			// (CommitInterval is set in consumer config)
		}
	}
}
