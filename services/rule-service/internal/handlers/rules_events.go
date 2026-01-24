// Package handlers provides HTTP handlers for the rule-service API.
package handlers

import (
	"context"
	"log/slog"
	"time"

	"rule-service/internal/database"
	"rule-service/internal/events"
)

// publishRuleChangedEvent publishes a rule.changed event after a successful DB operation.
// It logs errors but does not fail the operation if publishing fails.
func (h *Handlers) publishRuleChangedEvent(ctx context.Context, rule *database.Rule, action string) {
	changed := &events.RuleChanged{
		RuleID:        rule.RuleID,
		ClientID:      rule.ClientID,
		Action:        action,
		Version:       rule.Version,
		UpdatedAt:     rule.UpdatedAt.Unix(),
		SchemaVersion: SchemaVersion,
	}
	if err := h.producer.Publish(ctx, changed); err != nil {
		slog.Error("Failed to publish rule.changed event", "error", err, "rule_id", rule.RuleID)
		// Continue - the rule operation succeeded, event publishing failure can be handled separately
		return
	}
	// Track successful Kafka publish
	if h.metricsCollector != nil {
		h.metricsCollector.RecordPublished()
		h.metricsCollector.IncrementCustom("kafka_rule_" + action)
	}
}

// publishRuleDeletedEvent publishes a rule.changed event for a deleted rule.
// This is separate because DeleteRule needs to get the rule before deletion.
// Uses current time since rule.UpdatedAt may be stale after deletion.
func (h *Handlers) publishRuleDeletedEvent(ctx context.Context, rule *database.Rule) {
	changed := &events.RuleChanged{
		RuleID:        rule.RuleID,
		ClientID:      rule.ClientID,
		Action:        events.ActionDeleted,
		Version:       rule.Version,
		UpdatedAt:     time.Now().Unix(),
		SchemaVersion: SchemaVersion,
	}
	if err := h.producer.Publish(ctx, changed); err != nil {
		slog.Error("Failed to publish rule.changed event", "error", err, "rule_id", rule.RuleID)
		// Continue - the rule was deleted, event publishing failure can be handled separately
		return
	}
	// Track successful Kafka publish
	if h.metricsCollector != nil {
		h.metricsCollector.RecordPublished()
		h.metricsCollector.IncrementCustom("kafka_rule_" + events.ActionDeleted)
	}
}
