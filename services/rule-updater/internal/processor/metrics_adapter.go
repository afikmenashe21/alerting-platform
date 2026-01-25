// Package processor provides rule change processing orchestration.
package processor

import (
	"time"

	"github.com/afikmenashe/alerting-platform/pkg/metrics"
)

// metricsAdapter adapts *metrics.Collector to MetricsRecorder interface.
type metricsAdapter struct {
	collector *metrics.Collector
}

// NewMetricsAdapter wraps a metrics.Collector as a MetricsRecorder.
// If collector is nil, returns a no-op implementation.
func NewMetricsAdapter(collector *metrics.Collector) MetricsRecorder {
	if collector == nil {
		return NoopMetrics()
	}
	return &metricsAdapter{collector: collector}
}

func (m *metricsAdapter) RecordReceived() {
	m.collector.RecordReceived()
}

func (m *metricsAdapter) RecordProcessed(duration time.Duration) {
	m.collector.RecordProcessed(duration)
}

func (m *metricsAdapter) RecordPublished() {
	m.collector.RecordPublished()
}

func (m *metricsAdapter) RecordError() {
	m.collector.RecordError()
}

func (m *metricsAdapter) IncrementCustom(name string) {
	m.collector.IncrementCustom(name)
}
