// Package processor provides alert processing orchestration.
package processor

import "time"

// MetricsRecorder defines the interface for recording processing metrics.
// Using an interface allows for a no-op implementation when metrics are disabled,
// eliminating nil checks throughout the codebase.
type MetricsRecorder interface {
	RecordError()
	RecordProcessed(duration time.Duration)
	RecordPublished()
}

// NoOpMetrics is a no-op implementation of MetricsRecorder.
// Use this when metrics collection is disabled.
type NoOpMetrics struct{}

// Ensure NoOpMetrics implements MetricsRecorder.
var _ MetricsRecorder = (*NoOpMetrics)(nil)

func (NoOpMetrics) RecordError()                      {}
func (NoOpMetrics) RecordProcessed(time.Duration)     {}
func (NoOpMetrics) RecordPublished()                  {}
