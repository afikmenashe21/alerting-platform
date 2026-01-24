// Package processor provides rule change processing orchestration.
// It handles consuming rule.changed events and updating Redis snapshots.
package processor

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"rule-updater/internal/consumer"
	"rule-updater/internal/database"
	"rule-updater/internal/events"
	"rule-updater/internal/snapshot"

	"github.com/afikmenashe/alerting-platform/pkg/metrics"
)

// Processor orchestrates rule change processing and snapshot updates.
type Processor struct {
	consumer *consumer.Consumer
	db       *database.DB
	writer   *snapshot.Writer
	metrics  *metrics.Collector
}

// NewProcessor creates a new rule change processor (without metrics).
func NewProcessor(consumer *consumer.Consumer, db *database.DB, writer *snapshot.Writer) *Processor {
	return &Processor{
		consumer: consumer,
		db:       db,
		writer:   writer,
		metrics:  nil,
	}
}

// NewProcessorWithMetrics creates a processor with shared metrics collector.
func NewProcessorWithMetrics(consumer *consumer.Consumer, db *database.DB, writer *snapshot.Writer, m *metrics.Collector) *Processor {
	return &Processor{
		consumer: consumer,
		db:       db,
		writer:   writer,
		metrics:  m,
	}
}

// ProcessRuleChanges continuously reads rule.changed events from Kafka and updates
// the snapshot incrementally whenever a rule is created, updated, deleted, or disabled.
func (p *Processor) ProcessRuleChanges(ctx context.Context) error {
	slog.Info("Starting rule change processing loop")

	for {
		select {
		case <-ctx.Done():
			slog.Info("Rule change processing loop stopped")
			return nil
		default:
			// Read rule.changed event from Kafka
			ruleChanged, msg, err := p.consumer.ReadMessage(ctx)
			if err != nil {
				// Check if context was cancelled
				if ctx.Err() != nil {
					return nil
				}
				slog.Error("Failed to read rule.changed event", "error", err)
				// Continue processing other messages
				continue
			}

			if p.metrics != nil {
				p.metrics.RecordReceived()
			}

			startTime := time.Now()

			slog.Info("Received rule.changed event",
				"rule_id", ruleChanged.RuleID,
				"client_id", ruleChanged.ClientID,
				"action", ruleChanged.Action,
				"version", ruleChanged.Version,
			)

			// Apply incremental update directly to Redis using Lua scripts
			if err := p.applyRuleChange(ctx, ruleChanged); err != nil {
				slog.Error("Failed to apply rule change",
					"rule_id", ruleChanged.RuleID,
					"action", ruleChanged.Action,
					"error", err,
				)
				if p.metrics != nil {
					p.metrics.RecordError()
				}
				// Don't commit offset on error - Kafka will redeliver
				continue
			}

			if p.metrics != nil {
				p.metrics.RecordProcessed(time.Since(startTime))
				p.metrics.RecordPublished() // Track Redis write as "published"
				p.metrics.IncrementCustom("rules_" + string(ruleChanged.Action))
			}

			slog.Info("Rule change applied directly to Redis",
				"rule_id", ruleChanged.RuleID,
				"action", ruleChanged.Action,
			)

			// Commit offset only after successful snapshot update
			// This ensures at-least-once semantics: if we crash before commit, Kafka will redeliver
			if err := p.consumer.CommitMessage(ctx, msg); err != nil {
				slog.Error("Failed to commit offset",
					"rule_id", ruleChanged.RuleID,
					"action", ruleChanged.Action,
					"error", err,
				)
				// Continue processing - offset will be committed on next interval or retry
			}
		}
	}
}

// applyRuleChange applies a rule change event directly to Redis using Lua scripts.
// This avoids loading the entire snapshot into Go memory.
func (p *Processor) applyRuleChange(ctx context.Context, ruleChanged *events.RuleChanged) error {
	switch ruleChanged.Action {
	case events.ActionCreated, events.ActionUpdated:
		if p.db == nil {
			return fmt.Errorf("database is not configured")
		}
		if p.writer == nil {
			return fmt.Errorf("snapshot writer is not configured")
		}
		// For CREATED or UPDATED, fetch the rule from database and add/update it directly in Redis
		rule, err := p.db.GetRule(ctx, ruleChanged.RuleID)
		if err != nil {
			return fmt.Errorf("failed to get rule from database: %w", err)
		}

		// Add or update the rule directly in Redis using Lua script
		if err := p.writer.AddRuleDirect(ctx, rule); err != nil {
			return fmt.Errorf("failed to add/update rule in Redis: %w", err)
		}

		slog.Info("Rule added/updated directly in Redis",
			"rule_id", rule.RuleID,
			"enabled", rule.Enabled,
			"severity", rule.Severity,
			"source", rule.Source,
			"name", rule.Name,
		)

	case events.ActionDeleted, events.ActionDisabled:
		if p.writer == nil {
			return fmt.Errorf("snapshot writer is not configured")
		}
		// For DELETED or DISABLED, remove the rule directly from Redis using Lua script
		if err := p.writer.RemoveRuleDirect(ctx, ruleChanged.RuleID); err != nil {
			return fmt.Errorf("failed to remove rule from Redis: %w", err)
		}

		slog.Info("Rule removed directly from Redis",
			"rule_id", ruleChanged.RuleID,
			"action", ruleChanged.Action,
		)

	default:
		return fmt.Errorf("unknown action: %s", ruleChanged.Action)
	}

	return nil
}
