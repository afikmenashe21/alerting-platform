package processor

import (
	"context"
	"errors"
	"time"

	"aggregator/internal/events"

	"github.com/segmentio/kafka-go"
)

// FakeReader is a test fake for MessageReader.
type FakeReader struct {
	Messages   []*events.AlertMatched
	ReadErr    error
	CommitErr  error
	ReadIndex  int
	Committed  []kafka.Message
	ReadCalled int
}

func (f *FakeReader) ReadMessage(ctx context.Context) (*events.AlertMatched, *kafka.Message, error) {
	f.ReadCalled++
	if f.ReadErr != nil {
		return nil, nil, f.ReadErr
	}
	if f.ReadIndex >= len(f.Messages) {
		return nil, nil, errors.New("no more messages")
	}
	msg := f.Messages[f.ReadIndex]
	f.ReadIndex++
	return msg, &kafka.Message{}, nil
}

func (f *FakeReader) CommitMessage(ctx context.Context, msg *kafka.Message) error {
	if f.CommitErr != nil {
		return f.CommitErr
	}
	f.Committed = append(f.Committed, *msg)
	return nil
}

func (f *FakeReader) Close() error {
	return nil
}

// FakePublisher is a test fake for MessagePublisher.
type FakePublisher struct {
	Published   []*events.NotificationReady
	PublishErr  error
	PublishFunc func(ready *events.NotificationReady) error
}

func (f *FakePublisher) Publish(ctx context.Context, ready *events.NotificationReady) error {
	if f.PublishFunc != nil {
		return f.PublishFunc(ready)
	}
	if f.PublishErr != nil {
		return f.PublishErr
	}
	f.Published = append(f.Published, ready)
	return nil
}

func (f *FakePublisher) Close() error {
	return nil
}

// FakeStorage is a test fake for NotificationStorage.
type FakeStorage struct {
	InsertedNotifications []InsertCall
	InsertResult          *string
	InsertErr             error
	InsertFunc            func(clientID, alertID string) (*string, error)
}

type InsertCall struct {
	ClientID string
	AlertID  string
	Severity string
	Source   string
	Name     string
	Context  map[string]string
	RuleIDs  []string
}

func (f *FakeStorage) InsertNotificationIdempotent(
	ctx context.Context,
	clientID, alertID, severity, source, name string,
	context map[string]string,
	ruleIDs []string,
) (*string, error) {
	f.InsertedNotifications = append(f.InsertedNotifications, InsertCall{
		ClientID: clientID,
		AlertID:  alertID,
		Severity: severity,
		Source:   source,
		Name:     name,
		Context:  context,
		RuleIDs:  ruleIDs,
	})

	if f.InsertFunc != nil {
		return f.InsertFunc(clientID, alertID)
	}
	if f.InsertErr != nil {
		return nil, f.InsertErr
	}
	return f.InsertResult, nil
}

func (f *FakeStorage) Close() error {
	return nil
}

// FakeMetrics is a test fake for MetricsRecorder that tracks calls.
type FakeMetrics struct {
	ReceivedCount      int
	ProcessedCount     int
	PublishedCount     int
	ErrorCount         int
	CustomIncrements   map[string]int
	ProcessedLatencies []time.Duration
}

func NewFakeMetrics() *FakeMetrics {
	return &FakeMetrics{
		CustomIncrements: make(map[string]int),
	}
}

func (f *FakeMetrics) RecordReceived() {
	f.ReceivedCount++
}

func (f *FakeMetrics) RecordProcessed(latency time.Duration) {
	f.ProcessedCount++
	f.ProcessedLatencies = append(f.ProcessedLatencies, latency)
}

func (f *FakeMetrics) RecordPublished() {
	f.PublishedCount++
}

func (f *FakeMetrics) RecordError() {
	f.ErrorCount++
}

func (f *FakeMetrics) IncrementCustom(name string) {
	f.CustomIncrements[name]++
}
