package processor

import (
	"context"
	"log/slog"
	"time"

	"evaluator/internal/events"
)

// processResult contains the outcome of processing a single message.
type processResult struct {
	// allPublishesSucceeded is true if all matched alerts were published successfully.
	// When false, the message offset should NOT be committed to trigger redelivery.
	allPublishesSucceeded bool
	// publishedCount is the number of matched alerts successfully published.
	publishedCount int
}

// processOne handles a single alert: matches against rules and publishes results.
// Returns the processing result and records metrics.
//
// Responsibilities:
//   - Match alert against rules via matcher
//   - Publish one message per matching client
//   - Track success/failure for commit decision
//   - Record metrics (received, published, errors, latency)
func (p *Processor) processOne(ctx context.Context, alert *events.AlertNew) processResult {
	startTime := time.Now()

	// Match alert against rules
	matches := p.matcher.Match(alert.Severity, alert.Source, alert.Name)

	result := processResult{
		allPublishesSucceeded: true,
		publishedCount:        0,
	}

	if len(matches) == 0 {
		p.metrics.RecordProcessed(time.Since(startTime))
		p.metrics.IncrementCustom("alerts_unmatched")
		return result
	}

	// Publish one message per client_id
	for clientID, ruleIDs := range matches {
		matched := events.NewAlertMatched(alert, clientID, ruleIDs)

		if err := p.producer.Publish(ctx, matched); err != nil {
			slog.Error("Failed to publish matched alert",
				"alert_id", alert.AlertID,
				"client_id", clientID,
				"error", err,
			)
			p.metrics.RecordError()
			result.allPublishesSucceeded = false
			continue
		}

		result.publishedCount++
		p.metrics.RecordPublished()

		slog.Debug("Published matched alert",
			"alert_id", alert.AlertID,
			"client_id", clientID,
			"rule_ids", ruleIDs,
		)
	}

	p.metrics.RecordProcessed(time.Since(startTime))
	p.metrics.IncrementCustom("alerts_matched")

	return result
}
