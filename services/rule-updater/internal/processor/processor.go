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
	"github.com/segmentio/kafka-go"
)

// Processor orchestrates rule change processing and snapshot updates.
type Processor struct {
	consumer MessageConsumer
	db       RuleStore
	writer   SnapshotWriter
	metrics  MetricsRecorder
}

// Option configures a Processor.
type Option func(*Processor)

// WithMetrics sets the metrics recorder for the processor.
func WithMetrics(m MetricsRecorder) Option {
	return func(p *Processor) {
		if m != nil {
			p.metrics = m
		}
	}
}

// WithMetricsCollector sets a metrics.Collector as the metrics recorder.
func WithMetricsCollector(c *metrics.Collector) Option {
	return func(p *Processor) {
		p.metrics = NewMetricsAdapter(c)
	}
}

// New creates a new rule change processor with functional options.
func New(consumer MessageConsumer, db RuleStore, writer SnapshotWriter, opts ...Option) *Processor {
	p := &Processor{
		consumer: consumer,
		db:       db,
		writer:   writer,
		metrics:  NoopMetrics(),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// NewProcessor creates a new rule change processor (without metrics).
// Deprecated: Use New() with functional options instead.
func NewProcessor(consumer *consumer.Consumer, db *database.DB, writer *snapshot.Writer) *Processor {
	return New(consumer, db, writer)
}

// NewProcessorWithMetrics creates a processor with shared metrics collector.
// Deprecated: Use New() with WithMetricsCollector option instead.
func NewProcessorWithMetrics(consumer *consumer.Consumer, db *database.DB, writer *snapshot.Writer, m *metrics.Collector) *Processor {
	return New(consumer, db, writer, WithMetricsCollector(m))
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
			if err := p.processOneMessage(ctx); err != nil {
				if ctx.Err() != nil {
					return nil
				}
				// Log but continue - error already logged in processOneMessage
			}
		}
	}
}

// processOneMessage reads and processes a single message from Kafka.
// Returns an error if processing failed (message will not be committed).
func (p *Processor) processOneMessage(ctx context.Context) error {
	ruleChanged, msg, err := p.consumer.ReadMessage(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		slog.Error("Failed to read rule.changed event", "error", err)
		return err
	}

	p.metrics.RecordReceived()
	startTime := time.Now()

	slog.Info("Received rule.changed event",
		"rule_id", ruleChanged.RuleID,
		"client_id", ruleChanged.ClientID,
		"action", ruleChanged.Action,
		"version", ruleChanged.Version,
	)

	if err := p.applyRuleChange(ctx, ruleChanged); err != nil {
		slog.Error("Failed to apply rule change",
			"rule_id", ruleChanged.RuleID,
			"action", ruleChanged.Action,
			"error", err,
		)
		p.metrics.RecordError()
		return err
	}

	p.metrics.RecordProcessed(time.Since(startTime))
	p.metrics.RecordPublished()
	p.metrics.IncrementCustom("rules_" + ruleChanged.Action.String())

	slog.Info("Rule change applied directly to Redis",
		"rule_id", ruleChanged.RuleID,
		"action", ruleChanged.Action,
	)

	if err := p.commitMessage(ctx, msg, ruleChanged); err != nil {
		// Log but don't fail - offset will be committed on next interval or retry
		return nil
	}

	return nil
}

// commitMessage commits the Kafka message offset after successful processing.
func (p *Processor) commitMessage(ctx context.Context, msg *kafka.Message, ruleChanged *events.RuleChanged) error {
	if err := p.consumer.CommitMessage(ctx, msg); err != nil {
		slog.Error("Failed to commit offset",
			"rule_id", ruleChanged.RuleID,
			"action", ruleChanged.Action,
			"error", err,
		)
		return err
	}
	return nil
}

// applyRuleChange applies a rule change event directly to Redis using Lua scripts.
// This avoids loading the entire snapshot into Go memory.
func (p *Processor) applyRuleChange(ctx context.Context, ruleChanged *events.RuleChanged) error {
	if err := ruleChanged.Validate(); err != nil {
		return fmt.Errorf("invalid rule change event: %w", err)
	}

	if ruleChanged.Action.IsAdditive() {
		return p.applyAdditiveChange(ctx, ruleChanged)
	}

	if ruleChanged.Action.IsRemoval() {
		return p.applyRemovalChange(ctx, ruleChanged)
	}

	return fmt.Errorf("unknown action: %s", ruleChanged.Action)
}

// applyAdditiveChange handles CREATED and UPDATED actions.
func (p *Processor) applyAdditiveChange(ctx context.Context, ruleChanged *events.RuleChanged) error {
	if p.db == nil {
		return fmt.Errorf("database is not configured")
	}
	if p.writer == nil {
		return fmt.Errorf("snapshot writer is not configured")
	}

	rule, err := p.db.GetRule(ctx, ruleChanged.RuleID)
	if err != nil {
		return fmt.Errorf("failed to get rule from database: %w", err)
	}

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

	return nil
}

// applyRemovalChange handles DELETED and DISABLED actions.
func (p *Processor) applyRemovalChange(ctx context.Context, ruleChanged *events.RuleChanged) error {
	if p.writer == nil {
		return fmt.Errorf("snapshot writer is not configured")
	}

	if err := p.writer.RemoveRuleDirect(ctx, ruleChanged.RuleID); err != nil {
		return fmt.Errorf("failed to remove rule from Redis: %w", err)
	}

	slog.Info("Rule removed directly from Redis",
		"rule_id", ruleChanged.RuleID,
		"action", ruleChanged.Action,
	)

	return nil
}
