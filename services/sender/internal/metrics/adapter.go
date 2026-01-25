package metrics

import (
	"time"

	"github.com/afikmenashe/alerting-platform/pkg/metrics"
)

// CollectorAdapter adapts pkg/metrics.Collector to the Recorder interface.
type CollectorAdapter struct {
	collector *metrics.Collector
}

// NewCollectorAdapter wraps a metrics.Collector to implement Recorder.
func NewCollectorAdapter(collector *metrics.Collector) *CollectorAdapter {
	return &CollectorAdapter{collector: collector}
}

func (a *CollectorAdapter) RecordReceived() {
	a.collector.RecordReceived()
}

func (a *CollectorAdapter) RecordProcessed(latency time.Duration) {
	a.collector.RecordProcessed(latency)
}

func (a *CollectorAdapter) RecordPublished() {
	a.collector.RecordPublished()
}

func (a *CollectorAdapter) RecordError() {
	a.collector.RecordError()
}

func (a *CollectorAdapter) RecordSkipped() {
	a.collector.IncrementCustom("notifications_skipped")
}

func (a *CollectorAdapter) RecordFailed() {
	a.collector.IncrementCustom("notifications_failed")
}

func (a *CollectorAdapter) RecordSent() {
	a.collector.IncrementCustom("notifications_sent")
}

// Ensure CollectorAdapter implements Recorder
var _ Recorder = (*CollectorAdapter)(nil)
