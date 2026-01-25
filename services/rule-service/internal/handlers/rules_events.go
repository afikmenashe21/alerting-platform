// Package handlers provides HTTP handlers for the rule-service API.
package handlers

import (
	"context"
	"log/slog"
	"time"

	"rule-service/internal/database"
	"rule-service/internal/events"
)

// publishRuleEvent publishes a rule.changed event to Kafka.
// It logs errors but does not fail the operation if publishing fails.
// The updatedAt parameter allows customizing the timestamp (useful for deletions).
func (h *Handlers) publishRuleEvent(ctx context.Context, rule *database.Rule, action string, updatedAt int64) {
	changed := &events.RuleChanged{
		RuleID:        rule.RuleID,
		ClientID:      rule.ClientID,
		Action:        action,
		Version:       rule.Version,
		UpdatedAt:     updatedAt,
		SchemaVersion: SchemaVersion,
	}

	if err := h.producer.Publish(ctx, changed); err != nil {
		slog.Error("Failed to publish rule.changed event",
			"error", err,
			"rule_id", rule.RuleID,
			"action", action,
		)
		return
	}

	// Track successful Kafka publish using no-op pattern (no nil check needed)
	h.metrics.RecordPublished()
	h.metrics.IncrementCustom("kafka_rule_" + action)
}

// publishRuleChangedEvent publishes a rule.changed event after a successful DB operation.
// Uses the rule's UpdatedAt timestamp.
func (h *Handlers) publishRuleChangedEvent(ctx context.Context, rule *database.Rule, action string) {
	h.publishRuleEvent(ctx, rule, action, rule.UpdatedAt.Unix())
}

// publishRuleDeletedEvent publishes a rule.changed event for a deleted rule.
// Uses current time since rule.UpdatedAt may be stale after deletion.
func (h *Handlers) publishRuleDeletedEvent(ctx context.Context, rule *database.Rule) {
	h.publishRuleEvent(ctx, rule, events.ActionDeleted, time.Now().Unix())
}
