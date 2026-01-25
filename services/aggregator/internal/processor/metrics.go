// Package processor provides notification aggregation processing orchestration.
package processor

import "time"

// MetricsRecorder defines the metrics operations needed by the processor.
// This interface allows for dependency injection and testing with fakes.
type MetricsRecorder interface {
	RecordReceived()
	RecordProcessed(latency time.Duration)
	RecordPublished()
	RecordError()
	IncrementCustom(name string)
}

// NoOpMetrics is a null-object implementation of MetricsRecorder.
// It does nothing, eliminating the need for nil checks.
type NoOpMetrics struct{}

// Compile-time check that NoOpMetrics implements MetricsRecorder.
var _ MetricsRecorder = (*NoOpMetrics)(nil)

// RecordReceived does nothing.
func (n *NoOpMetrics) RecordReceived() {}

// RecordProcessed does nothing.
func (n *NoOpMetrics) RecordProcessed(_ time.Duration) {}

// RecordPublished does nothing.
func (n *NoOpMetrics) RecordPublished() {}

// RecordError does nothing.
func (n *NoOpMetrics) RecordError() {}

// IncrementCustom does nothing.
func (n *NoOpMetrics) IncrementCustom(_ string) {}
