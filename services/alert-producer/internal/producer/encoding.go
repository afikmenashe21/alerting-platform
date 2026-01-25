// Package producer provides a Kafka producer wrapper for publishing alerts.
package producer

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"alert-producer/internal/generator"

	pbalerts "github.com/afikmenashe/alerting-platform/pkg/proto/alerts"
	pbcommon "github.com/afikmenashe/alerting-platform/pkg/proto/common"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

// severityFromString converts a severity string to a protobuf Severity enum.
// Returns Severity_UNSPECIFIED for unknown values.
func severityFromString(s string) pbcommon.Severity {
	switch strings.ToUpper(s) {
	case "LOW":
		return pbcommon.Severity_LOW
	case "MEDIUM":
		return pbcommon.Severity_MEDIUM
	case "HIGH":
		return pbcommon.Severity_HIGH
	case "CRITICAL":
		return pbcommon.Severity_CRITICAL
	default:
		return pbcommon.Severity_UNSPECIFIED
	}
}

// alertToProto converts an Alert to its protobuf representation.
func alertToProto(alert *generator.Alert) *pbalerts.AlertNew {
	return &pbalerts.AlertNew{
		AlertId:       alert.AlertID,
		SchemaVersion: int32(alert.SchemaVersion),
		EventTs:       alert.EventTS,
		Severity:      severityFromString(alert.Severity),
		Source:        alert.Source,
		Name:          alert.Name,
		Context:       alert.Context,
	}
}

// encodeAlert serializes an alert to protobuf bytes.
func encodeAlert(alert *generator.Alert) ([]byte, error) {
	pb := alertToProto(alert)
	payload, err := proto.Marshal(pb)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal alert: %w", err)
	}
	return payload, nil
}

// buildKafkaMessage creates a Kafka message from an alert and its encoded payload.
// The message is keyed by a hash of alert_id for even partition distribution.
func buildKafkaMessage(alert *generator.Alert, payload []byte) kafka.Message {
	return kafka.Message{
		Key:   hashAlertID(alert.AlertID),
		Value: payload,
		Headers: []kafka.Header{
			{Key: "content-type", Value: []byte("application/x-protobuf")},
			{Key: "schema_version", Value: []byte(fmt.Sprintf("%d", alert.SchemaVersion))},
			{Key: "severity", Value: []byte(alert.Severity)},
		},
		Time: time.Unix(alert.EventTS, 0),
	}
}

// hashAlertID creates a deterministic hash of the alert_id for partition key.
// Returns the first 16 bytes of SHA256 for good distribution with reasonable size.
func hashAlertID(alertID string) []byte {
	hash := sha256.Sum256([]byte(alertID))
	return hash[:16]
}
