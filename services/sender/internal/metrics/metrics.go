// Package metrics provides metrics recording interfaces for the sender service.
// It uses the null object pattern to avoid nil checks throughout the codebase.
package metrics

import "time"

// Recorder defines the interface for recording sender metrics.
// Implementations can record to various backends (Redis, Prometheus, etc.)
type Recorder interface {
	// RecordReceived increments the count of received messages.
	RecordReceived()

	// RecordProcessed records a successfully processed message with its latency.
	RecordProcessed(latency time.Duration)

	// RecordPublished increments the count of published/sent notifications.
	RecordPublished()

	// RecordError increments the error counter.
	RecordError()

	// RecordSkipped increments the count of skipped notifications (already processed).
	RecordSkipped()

	// RecordFailed increments the count of failed notifications (DLQ).
	RecordFailed()

	// RecordSent increments the count of successfully sent notifications.
	RecordSent()
}

// NoOp is a no-op implementation of Recorder that discards all metrics.
// Use this when metrics collection is not configured.
type NoOp struct{}

// NewNoOp creates a new no-op metrics recorder.
func NewNoOp() *NoOp {
	return &NoOp{}
}

func (n *NoOp) RecordReceived()                   {}
func (n *NoOp) RecordProcessed(_ time.Duration)   {}
func (n *NoOp) RecordPublished()                  {}
func (n *NoOp) RecordError()                      {}
func (n *NoOp) RecordSkipped()                    {}
func (n *NoOp) RecordFailed()                     {}
func (n *NoOp) RecordSent()                       {}

// Ensure NoOp implements Recorder
var _ Recorder = (*NoOp)(nil)
