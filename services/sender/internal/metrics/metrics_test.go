package metrics

import (
	"testing"
	"time"
)

func TestNoOp_ImplementsRecorder(t *testing.T) {
	var _ Recorder = (*NoOp)(nil)
}

func TestNoOp_AllMethodsWork(t *testing.T) {
	noop := NewNoOp()

	// All these should not panic
	noop.RecordReceived()
	noop.RecordProcessed(time.Second)
	noop.RecordPublished()
	noop.RecordError()
	noop.RecordSkipped()
	noop.RecordFailed()
	noop.RecordSent()
}

func TestNewNoOp(t *testing.T) {
	noop := NewNoOp()
	if noop == nil {
		t.Error("NewNoOp() returned nil")
	}
}
