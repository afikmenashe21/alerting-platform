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
			alert, msg, err := p.consumer.ReadMessage(ctx)
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

			// Track if all publishes succeeded for commit decision
			allPublishesSucceeded := true

			// Publish one message per client_id
			if len(matches) > 0 {
				for clientID, ruleIDs := range matches {
					// Build matched alert event for this client
					matched := events.NewAlertMatched(alert, clientID, ruleIDs)

					// Publish message for this client
					if err := p.producer.Publish(ctx, matched); err != nil {
						slog.Error("Failed to publish matched alert",
							"alert_id", alert.AlertID,
							"client_id", clientID,
							"error", err,
						)
						allPublishesSucceeded = false
						// Continue processing other clients
						continue
					}

					slog.Info("Published matched alert",
						"alert_id", alert.AlertID,
						"client_id", clientID,
						"rule_ids", ruleIDs,
					)
				}
			} else {
				slog.Debug("No rules matched alert",
					"alert_id", alert.AlertID,
					"severity", alert.Severity,
					"source", alert.Source,
					"name", alert.Name,
				)
			}

			// Commit offset only after all publishes succeeded (or no matches)
			// This ensures at-least-once semantics: if we crash before commit, Kafka will redeliver
			if allPublishesSucceeded {
				if err := p.consumer.CommitMessage(ctx, msg); err != nil {
					slog.Error("Failed to commit offset",
						"alert_id", alert.AlertID,
						"error", err,
					)
				}
			} else {
				slog.Warn("Skipping offset commit due to publish failures, message will be redelivered",
					"alert_id", alert.AlertID,
				)
			}
		}
	}
}
