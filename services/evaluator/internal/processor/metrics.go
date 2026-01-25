// Package processor provides alert evaluation processing orchestration.
package processor

import "time"

// Metrics defines the interface for recording processor metrics.
// Implementations must be safe for concurrent use.
type Metrics interface {
	// RecordReceived increments the count of alerts received from Kafka.
	RecordReceived()
	// RecordPublished increments the count of matched alerts published.
	RecordPublished()
	// RecordError increments the count of processing errors.
	RecordError()
	// RecordProcessed records the processing duration for a single alert.
	RecordProcessed(duration time.Duration)
	// IncrementCustom increments a custom counter by name.
	IncrementCustom(name string)
	// AddCustom adds a value to a custom counter by name.
	AddCustom(name string, value uint64)
}

// NoOpMetrics is a no-op implementation of Metrics.
// Use this when metrics collection is disabled.
type NoOpMetrics struct{}

func (NoOpMetrics) RecordReceived()                {}
func (NoOpMetrics) RecordPublished()               {}
func (NoOpMetrics) RecordError()                   {}
func (NoOpMetrics) RecordProcessed(time.Duration)  {}
func (NoOpMetrics) IncrementCustom(string)         {}
func (NoOpMetrics) AddCustom(string, uint64)       {}

// collectorAdapter adapts *metrics.Collector to the Metrics interface.
// This keeps the processor package decoupled from the concrete metrics implementation.
type collectorAdapter struct {
	c metricsCollector
}

// metricsCollector is the minimal interface we need from *metrics.Collector.
// This avoids importing the metrics package in the interface definition.
type metricsCollector interface {
	RecordReceived()
	RecordPublished()
	RecordError()
	RecordProcessed(duration time.Duration)
	IncrementCustom(name string)
	AddCustom(name string, value uint64)
}

func (a *collectorAdapter) RecordReceived()                   { a.c.RecordReceived() }
func (a *collectorAdapter) RecordPublished()                  { a.c.RecordPublished() }
func (a *collectorAdapter) RecordError()                      { a.c.RecordError() }
func (a *collectorAdapter) RecordProcessed(d time.Duration)   { a.c.RecordProcessed(d) }
func (a *collectorAdapter) IncrementCustom(name string)       { a.c.IncrementCustom(name) }
func (a *collectorAdapter) AddCustom(name string, val uint64) { a.c.AddCustom(name, val) }

// wrapMetrics wraps a metricsCollector (or nil) into a Metrics interface.
// If c is nil, returns NoOpMetrics to avoid nil checks throughout the code.
func wrapMetrics(c metricsCollector) Metrics {
	if c == nil {
		return NoOpMetrics{}
	}
	return &collectorAdapter{c: c}
}
