package processor

import (
	"context"
	"errors"
	"testing"
	"time"

	"rule-updater/internal/database"
	"rule-updater/internal/events"
)

func TestNew(t *testing.T) {
	consumer := newFakeConsumer()
	store := newFakeRuleStore()
	writer := newFakeSnapshotWriter()

	p := New(consumer, store, writer)

	if p == nil {
		t.Fatal("New() returned nil")
	}
	if p.consumer != consumer {
		t.Error("New() consumer not set correctly")
	}
	if p.db != store {
		t.Error("New() db not set correctly")
	}
	if p.writer != writer {
		t.Error("New() writer not set correctly")
	}
	// Verify metrics defaults to no-op (not nil)
	if p.metrics == nil {
		t.Error("New() metrics should default to no-op, not nil")
	}
}

func TestNew_WithMetrics(t *testing.T) {
	consumer := newFakeConsumer()
	store := newFakeRuleStore()
	writer := newFakeSnapshotWriter()
	metrics := newFakeMetrics()

	p := New(consumer, store, writer, WithMetrics(metrics))

	if p.metrics != metrics {
		t.Error("WithMetrics() option not applied correctly")
	}
}

func TestProcessRuleChanges_ContextCancellation(t *testing.T) {
	consumer := newFakeConsumer()
	store := newFakeRuleStore()
	writer := newFakeSnapshotWriter()

	p := New(consumer, store, writer)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := p.ProcessRuleChanges(ctx)
	if err != nil {
		t.Errorf("ProcessRuleChanges() error = %v, want nil", err)
	}
}

func TestApplyRuleChange_Created(t *testing.T) {
	store := newFakeRuleStore()
	writer := newFakeSnapshotWriter()
	metrics := newFakeMetrics()

	rule := &database.Rule{
		RuleID:   "rule-1",
		ClientID: "client-1",
		Severity: "critical",
		Source:   "test-source",
		Name:     "test-rule",
		Enabled:  true,
	}
	store.AddRule(rule)

	p := New(nil, store, writer, WithMetrics(metrics))

	ruleChanged := &events.RuleChanged{
		RuleID:   "rule-1",
		ClientID: "client-1",
		Action:   events.ActionCreated,
		Version:  1,
	}

	err := p.applyRuleChange(context.Background(), ruleChanged)
	if err != nil {
		t.Errorf("applyRuleChange() error = %v, want nil", err)
	}

	if len(writer.addedRules) != 1 {
		t.Errorf("expected 1 added rule, got %d", len(writer.addedRules))
	}
	if writer.addedRules[0].RuleID != "rule-1" {
		t.Errorf("expected rule-1, got %s", writer.addedRules[0].RuleID)
	}
}

func TestApplyRuleChange_Updated(t *testing.T) {
	store := newFakeRuleStore()
	writer := newFakeSnapshotWriter()

	rule := &database.Rule{
		RuleID:   "rule-2",
		ClientID: "client-1",
		Severity: "warning",
		Source:   "test-source",
		Name:     "updated-rule",
		Enabled:  true,
	}
	store.AddRule(rule)

	p := New(nil, store, writer)

	ruleChanged := &events.RuleChanged{
		RuleID:   "rule-2",
		ClientID: "client-1",
		Action:   events.ActionUpdated,
		Version:  2,
	}

	err := p.applyRuleChange(context.Background(), ruleChanged)
	if err != nil {
		t.Errorf("applyRuleChange() error = %v, want nil", err)
	}

	if len(writer.addedRules) != 1 {
		t.Errorf("expected 1 added rule, got %d", len(writer.addedRules))
	}
}

func TestApplyRuleChange_Deleted(t *testing.T) {
	writer := newFakeSnapshotWriter()

	p := New(nil, nil, writer)

	ruleChanged := &events.RuleChanged{
		RuleID:   "rule-3",
		ClientID: "client-1",
		Action:   events.ActionDeleted,
		Version:  1,
	}

	err := p.applyRuleChange(context.Background(), ruleChanged)
	if err != nil {
		t.Errorf("applyRuleChange() error = %v, want nil", err)
	}

	if len(writer.removedRules) != 1 {
		t.Errorf("expected 1 removed rule, got %d", len(writer.removedRules))
	}
	if writer.removedRules[0] != "rule-3" {
		t.Errorf("expected rule-3, got %s", writer.removedRules[0])
	}
}

func TestApplyRuleChange_Disabled(t *testing.T) {
	writer := newFakeSnapshotWriter()

	p := New(nil, nil, writer)

	ruleChanged := &events.RuleChanged{
		RuleID:   "rule-4",
		ClientID: "client-1",
		Action:   events.ActionDisabled,
		Version:  1,
	}

	err := p.applyRuleChange(context.Background(), ruleChanged)
	if err != nil {
		t.Errorf("applyRuleChange() error = %v, want nil", err)
	}

	if len(writer.removedRules) != 1 {
		t.Errorf("expected 1 removed rule, got %d", len(writer.removedRules))
	}
}

func TestApplyRuleChange_UnknownAction(t *testing.T) {
	p := New(nil, nil, nil)

	ruleChanged := &events.RuleChanged{
		RuleID:   "rule-5",
		ClientID: "client-1",
		Action:   events.Action("UNKNOWN"),
		Version:  1,
	}

	err := p.applyRuleChange(context.Background(), ruleChanged)
	if err == nil {
		t.Error("applyRuleChange() with unknown action expected error, got nil")
	}
}

func TestApplyRuleChange_EmptyRuleID(t *testing.T) {
	p := New(nil, nil, nil)

	ruleChanged := &events.RuleChanged{
		RuleID:   "",
		ClientID: "client-1",
		Action:   events.ActionCreated,
		Version:  1,
	}

	err := p.applyRuleChange(context.Background(), ruleChanged)
	if err == nil {
		t.Error("applyRuleChange() with empty rule_id expected error, got nil")
	}
}

func TestApplyRuleChange_DBError(t *testing.T) {
	store := newFakeRuleStore()
	store.SetGetError(errors.New("db connection failed"))
	writer := newFakeSnapshotWriter()

	p := New(nil, store, writer)

	ruleChanged := &events.RuleChanged{
		RuleID:   "rule-6",
		ClientID: "client-1",
		Action:   events.ActionCreated,
		Version:  1,
	}

	err := p.applyRuleChange(context.Background(), ruleChanged)
	if err == nil {
		t.Error("applyRuleChange() with DB error expected error, got nil")
	}
}

func TestApplyRuleChange_WriterError(t *testing.T) {
	store := newFakeRuleStore()
	store.AddRule(&database.Rule{
		RuleID:   "rule-7",
		ClientID: "client-1",
		Enabled:  true,
	})
	writer := newFakeSnapshotWriter()
	writer.SetAddError(errors.New("redis connection failed"))

	p := New(nil, store, writer)

	ruleChanged := &events.RuleChanged{
		RuleID:   "rule-7",
		ClientID: "client-1",
		Action:   events.ActionCreated,
		Version:  1,
	}

	err := p.applyRuleChange(context.Background(), ruleChanged)
	if err == nil {
		t.Error("applyRuleChange() with writer error expected error, got nil")
	}
}

func TestProcessOneMessage_Success(t *testing.T) {
	consumer := newFakeConsumer()
	store := newFakeRuleStore()
	writer := newFakeSnapshotWriter()
	metrics := newFakeMetrics()

	rule := &database.Rule{
		RuleID:   "rule-8",
		ClientID: "client-1",
		Severity: "info",
		Source:   "test",
		Name:     "test-rule",
		Enabled:  true,
	}
	store.AddRule(rule)

	consumer.AddMessage(&events.RuleChanged{
		RuleID:   "rule-8",
		ClientID: "client-1",
		Action:   events.ActionCreated,
		Version:  1,
	})

	p := New(consumer, store, writer, WithMetrics(metrics))

	err := p.processOneMessage(context.Background())
	if err != nil {
		t.Errorf("processOneMessage() error = %v, want nil", err)
	}

	// Verify metrics were recorded
	if metrics.receivedCalls != 1 {
		t.Errorf("expected 1 RecordReceived call, got %d", metrics.receivedCalls)
	}
	if metrics.processedCalls != 1 {
		t.Errorf("expected 1 RecordProcessed call, got %d", metrics.processedCalls)
	}
	if metrics.publishedCalls != 1 {
		t.Errorf("expected 1 RecordPublished call, got %d", metrics.publishedCalls)
	}
	if metrics.customCalls["rules_CREATED"] != 1 {
		t.Errorf("expected 1 rules_CREATED metric, got %d", metrics.customCalls["rules_CREATED"])
	}

	// Verify commit was called
	if consumer.commitCalls != 1 {
		t.Errorf("expected 1 commit call, got %d", consumer.commitCalls)
	}
}

func TestProcessOneMessage_ApplyError(t *testing.T) {
	consumer := newFakeConsumer()
	writer := newFakeSnapshotWriter()
	metrics := newFakeMetrics()

	consumer.AddMessage(&events.RuleChanged{
		RuleID:   "rule-9",
		ClientID: "client-1",
		Action:   events.ActionCreated, // Will fail - no DB configured
		Version:  1,
	})

	// No DB configured - will fail on CREATED action
	p := New(consumer, nil, writer, WithMetrics(metrics))

	err := p.processOneMessage(context.Background())
	if err == nil {
		t.Error("processOneMessage() expected error, got nil")
	}

	// Verify error metric was recorded
	if metrics.errorCalls != 1 {
		t.Errorf("expected 1 RecordError call, got %d", metrics.errorCalls)
	}

	// Verify commit was NOT called (message not committed on error)
	if consumer.commitCalls != 0 {
		t.Errorf("expected 0 commit calls on error, got %d", consumer.commitCalls)
	}
}

func TestProcessOneMessage_ReadError(t *testing.T) {
	consumer := newFakeConsumer()
	consumer.SetReadError(errors.New("kafka read failed"))

	p := New(consumer, nil, nil)

	err := p.processOneMessage(context.Background())
	if err == nil {
		t.Error("processOneMessage() with read error expected error, got nil")
	}
}

func TestNoopMetrics(t *testing.T) {
	m := NoopMetrics()

	// These should not panic
	m.RecordReceived()
	m.RecordProcessed(time.Second)
	m.RecordPublished()
	m.RecordError()
	m.IncrementCustom("test")
}

func TestAction_IsAdditive(t *testing.T) {
	tests := []struct {
		action   events.Action
		expected bool
	}{
		{events.ActionCreated, true},
		{events.ActionUpdated, true},
		{events.ActionDeleted, false},
		{events.ActionDisabled, false},
		{events.Action("UNKNOWN"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			if got := tt.action.IsAdditive(); got != tt.expected {
				t.Errorf("Action(%s).IsAdditive() = %v, want %v", tt.action, got, tt.expected)
			}
		})
	}
}

func TestAction_IsRemoval(t *testing.T) {
	tests := []struct {
		action   events.Action
		expected bool
	}{
		{events.ActionCreated, false},
		{events.ActionUpdated, false},
		{events.ActionDeleted, true},
		{events.ActionDisabled, true},
		{events.Action("UNKNOWN"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			if got := tt.action.IsRemoval(); got != tt.expected {
				t.Errorf("Action(%s).IsRemoval() = %v, want %v", tt.action, got, tt.expected)
			}
		})
	}
}
