// Package processor provides rule change event handling for the evaluator.
package processor

import (
	"context"
	"log/slog"

	"evaluator/internal/reloader"
	"evaluator/internal/ruleconsumer"
)

// RuleHandler handles rule.changed events and triggers immediate reloads.
type RuleHandler struct {
	consumer *ruleconsumer.Consumer
	reload   *reloader.Reloader
}

// NewRuleHandler creates a new rule change handler.
func NewRuleHandler(consumer *ruleconsumer.Consumer, reload *reloader.Reloader) *RuleHandler {
	return &RuleHandler{
		consumer: consumer,
		reload:   reload,
	}
}

// HandleRuleChanged consumes rule.changed events and triggers immediate reloads.
func (h *RuleHandler) HandleRuleChanged(ctx context.Context) {
	slog.Info("Starting rule.changed event handler")

	for {
		select {
		case <-ctx.Done():
			slog.Info("Rule.changed event handler stopped")
			return
		default:
			ruleChanged, err := h.consumer.ReadMessage(ctx)
			if err != nil {
				// Check if context was cancelled
				if ctx.Err() != nil {
					return
				}
				slog.Error("Failed to read rule.changed event", "error", err)
				// Continue processing other messages
				continue
			}

			slog.Info("Received rule.changed event",
				"rule_id", ruleChanged.RuleID,
				"client_id", ruleChanged.ClientID,
				"action", ruleChanged.Action,
				"version", ruleChanged.Version,
			)

			// Trigger immediate reload from Redis snapshot
			// The rule-updater should have already updated the snapshot
			if err := h.reload.ReloadNow(ctx); err != nil {
				slog.Error("Failed to reload rules after rule.changed event",
					"rule_id", ruleChanged.RuleID,
					"action", ruleChanged.Action,
					"error", err,
				)
				// Continue - polling will catch up eventually
			} else {
				slog.Info("Rules reloaded after rule.changed event",
					"rule_id", ruleChanged.RuleID,
					"action", ruleChanged.Action,
				)
			}
		}
	}
}
