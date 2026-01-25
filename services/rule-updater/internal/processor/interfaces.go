// Package processor provides rule change processing orchestration.
package processor

import (
	"context"
	"time"

	"rule-updater/internal/database"
	"rule-updater/internal/events"

	"github.com/segmentio/kafka-go"
)

// MessageConsumer reads and commits Kafka messages.
type MessageConsumer interface {
	ReadMessage(ctx context.Context) (*events.RuleChanged, *kafka.Message, error)
	CommitMessage(ctx context.Context, msg *kafka.Message) error
	Close() error
}

// RuleStore provides access to rules in the database.
type RuleStore interface {
	GetRule(ctx context.Context, ruleID string) (*database.Rule, error)
}

// SnapshotWriter writes rule changes to the snapshot store.
type SnapshotWriter interface {
	AddRuleDirect(ctx context.Context, rule *database.Rule) error
	RemoveRuleDirect(ctx context.Context, ruleID string) error
}

// MetricsRecorder records processing metrics.
type MetricsRecorder interface {
	RecordReceived()
	RecordProcessed(duration time.Duration)
	RecordPublished()
	RecordError()
	IncrementCustom(name string)
}

// noopMetrics is a no-op implementation of MetricsRecorder.
// This avoids scattered nil checks throughout the code.
type noopMetrics struct{}

func (noopMetrics) RecordReceived()                   {}
func (noopMetrics) RecordProcessed(time.Duration)    {}
func (noopMetrics) RecordPublished()                 {}
func (noopMetrics) RecordError()                     {}
func (noopMetrics) IncrementCustom(string)           {}

// NoopMetrics returns a no-op metrics recorder.
func NoopMetrics() MetricsRecorder {
	return noopMetrics{}
}
