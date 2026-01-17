package producer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"alert-producer/internal/generator"
)

// MockProducer is a mock implementation that logs alerts instead of publishing to Kafka.
// Useful for testing without a Kafka instance.
type MockProducer struct {
	topic string
}

// Ensure MockProducer implements AlertPublisher interface
var _ AlertPublisher = (*MockProducer)(nil)

// NewMock creates a new mock producer that logs alerts instead of publishing to Kafka.
func NewMock(topic string) *MockProducer {
	slog.Info("Using mock producer (no Kafka connection)",
		"topic", topic,
		"note", "Alerts will be logged but not published to Kafka",
	)
	return &MockProducer{
		topic: topic,
	}
}

// Publish logs the alert as JSON instead of publishing to Kafka.
func (p *MockProducer) Publish(ctx context.Context, alert *generator.Alert) error {
	// Serialize alert to JSON for logging
	payload, err := json.Marshal(alert)
	if err != nil {
		slog.Error("Failed to marshal alert in mock producer",
			"alert_id", alert.AlertID,
			"error", err,
		)
		return fmt.Errorf("failed to marshal alert: %w", err)
	}

	// Log the alert (simulating successful publish)
	slog.Info("Mock publish (alert logged, not sent to Kafka)",
		"topic", p.topic,
		"alert_id", alert.AlertID,
		"severity", alert.Severity,
		"source", alert.Source,
		"name", alert.Name,
		"alert_json", string(payload),
	)

	return nil
}

// Close is a no-op for the mock producer.
func (p *MockProducer) Close() error {
	slog.Info("Mock producer closed", "topic", p.topic)
	return nil
}
