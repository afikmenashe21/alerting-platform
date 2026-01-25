package processor

import (
	"context"
	"errors"
	"time"

	"rule-updater/internal/database"
	"rule-updater/internal/events"

	"github.com/segmentio/kafka-go"
)

// fakeConsumer is a test fake for MessageConsumer.
type fakeConsumer struct {
	messages     []*events.RuleChanged
	kafkaMessages []*kafka.Message
	readIndex    int
	readErr      error
	commitErr    error
	commitCalls  int
}

func newFakeConsumer() *fakeConsumer {
	return &fakeConsumer{}
}

func (f *fakeConsumer) AddMessage(rc *events.RuleChanged) {
	f.messages = append(f.messages, rc)
	f.kafkaMessages = append(f.kafkaMessages, &kafka.Message{
		Topic:     "rule.changed",
		Partition: 0,
		Offset:    int64(len(f.messages)),
		Value:     []byte("test"),
	})
}

func (f *fakeConsumer) SetReadError(err error) {
	f.readErr = err
}

func (f *fakeConsumer) SetCommitError(err error) {
	f.commitErr = err
}

func (f *fakeConsumer) ReadMessage(ctx context.Context) (*events.RuleChanged, *kafka.Message, error) {
	if f.readErr != nil {
		return nil, nil, f.readErr
	}
	if f.readIndex >= len(f.messages) {
		// Block until context is cancelled (simulates waiting for message)
		<-ctx.Done()
		return nil, nil, ctx.Err()
	}
	rc := f.messages[f.readIndex]
	msg := f.kafkaMessages[f.readIndex]
	f.readIndex++
	return rc, msg, nil
}

func (f *fakeConsumer) CommitMessage(ctx context.Context, msg *kafka.Message) error {
	f.commitCalls++
	return f.commitErr
}

func (f *fakeConsumer) Close() error {
	return nil
}

// fakeRuleStore is a test fake for RuleStore.
type fakeRuleStore struct {
	rules   map[string]*database.Rule
	getErr  error
	getCalls int
}

func newFakeRuleStore() *fakeRuleStore {
	return &fakeRuleStore{
		rules: make(map[string]*database.Rule),
	}
}

func (f *fakeRuleStore) AddRule(rule *database.Rule) {
	f.rules[rule.RuleID] = rule
}

func (f *fakeRuleStore) SetGetError(err error) {
	f.getErr = err
}

func (f *fakeRuleStore) GetRule(ctx context.Context, ruleID string) (*database.Rule, error) {
	f.getCalls++
	if f.getErr != nil {
		return nil, f.getErr
	}
	rule, ok := f.rules[ruleID]
	if !ok {
		return nil, errors.New("rule not found")
	}
	return rule, nil
}

// fakeSnapshotWriter is a test fake for SnapshotWriter.
type fakeSnapshotWriter struct {
	addedRules    []*database.Rule
	removedRules  []string
	addErr        error
	removeErr     error
}

func newFakeSnapshotWriter() *fakeSnapshotWriter {
	return &fakeSnapshotWriter{}
}

func (f *fakeSnapshotWriter) SetAddError(err error) {
	f.addErr = err
}

func (f *fakeSnapshotWriter) SetRemoveError(err error) {
	f.removeErr = err
}

func (f *fakeSnapshotWriter) AddRuleDirect(ctx context.Context, rule *database.Rule) error {
	if f.addErr != nil {
		return f.addErr
	}
	f.addedRules = append(f.addedRules, rule)
	return nil
}

func (f *fakeSnapshotWriter) RemoveRuleDirect(ctx context.Context, ruleID string) error {
	if f.removeErr != nil {
		return f.removeErr
	}
	f.removedRules = append(f.removedRules, ruleID)
	return nil
}

// fakeMetrics is a test fake for MetricsRecorder.
type fakeMetrics struct {
	receivedCalls  int
	processedCalls int
	publishedCalls int
	errorCalls     int
	customCalls    map[string]int
}

func newFakeMetrics() *fakeMetrics {
	return &fakeMetrics{
		customCalls: make(map[string]int),
	}
}

func (f *fakeMetrics) RecordReceived() {
	f.receivedCalls++
}

func (f *fakeMetrics) RecordProcessed(duration time.Duration) {
	f.processedCalls++
}

func (f *fakeMetrics) RecordPublished() {
	f.publishedCalls++
}

func (f *fakeMetrics) RecordError() {
	f.errorCalls++
}

func (f *fakeMetrics) IncrementCustom(name string) {
	f.customCalls[name]++
}
