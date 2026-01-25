package producer

import (
	"testing"
	"time"

	"alert-producer/internal/generator"

	pbcommon "github.com/afikmenashe/alerting-platform/pkg/proto/common"
)

func TestSeverityFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected pbcommon.Severity
	}{
		{"low", "low", pbcommon.Severity_LOW},
		{"LOW", "LOW", pbcommon.Severity_LOW},
		{"medium", "medium", pbcommon.Severity_MEDIUM},
		{"MEDIUM", "MEDIUM", pbcommon.Severity_MEDIUM},
		{"high", "high", pbcommon.Severity_HIGH},
		{"HIGH", "HIGH", pbcommon.Severity_HIGH},
		{"critical", "critical", pbcommon.Severity_CRITICAL},
		{"CRITICAL", "CRITICAL", pbcommon.Severity_CRITICAL},
		{"unknown", "unknown", pbcommon.Severity_UNSPECIFIED},
		{"empty", "", pbcommon.Severity_UNSPECIFIED},
		{"mixed case", "HiGh", pbcommon.Severity_HIGH},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := severityFromString(tt.input)
			if result != tt.expected {
				t.Errorf("severityFromString(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAlertToProto(t *testing.T) {
	alert := &generator.Alert{
		AlertID:       "test-id-123",
		SchemaVersion: 1,
		EventTS:       1234567890,
		Severity:      "HIGH",
		Source:        "api",
		Name:          "timeout",
		Context:       map[string]string{"key": "value"},
	}

	pb := alertToProto(alert)

	if pb.AlertId != alert.AlertID {
		t.Errorf("AlertId = %v, want %v", pb.AlertId, alert.AlertID)
	}
	if pb.SchemaVersion != int32(alert.SchemaVersion) {
		t.Errorf("SchemaVersion = %v, want %v", pb.SchemaVersion, alert.SchemaVersion)
	}
	if pb.EventTs != alert.EventTS {
		t.Errorf("EventTs = %v, want %v", pb.EventTs, alert.EventTS)
	}
	if pb.Severity != pbcommon.Severity_HIGH {
		t.Errorf("Severity = %v, want HIGH", pb.Severity)
	}
	if pb.Source != alert.Source {
		t.Errorf("Source = %v, want %v", pb.Source, alert.Source)
	}
	if pb.Name != alert.Name {
		t.Errorf("Name = %v, want %v", pb.Name, alert.Name)
	}
	if pb.Context["key"] != "value" {
		t.Errorf("Context[key] = %v, want value", pb.Context["key"])
	}
}

func TestEncodeAlert(t *testing.T) {
	alert := &generator.Alert{
		AlertID:       "test-id-123",
		SchemaVersion: 1,
		EventTS:       1234567890,
		Severity:      "HIGH",
		Source:        "api",
		Name:          "timeout",
	}

	payload, err := encodeAlert(alert)
	if err != nil {
		t.Fatalf("encodeAlert failed: %v", err)
	}
	if len(payload) == 0 {
		t.Error("encodeAlert returned empty payload")
	}
}

func TestBuildKafkaMessage(t *testing.T) {
	alert := &generator.Alert{
		AlertID:       "test-id-123",
		SchemaVersion: 1,
		EventTS:       1234567890,
		Severity:      "HIGH",
		Source:        "api",
		Name:          "timeout",
	}
	payload := []byte("test-payload")

	msg := buildKafkaMessage(alert, payload)

	if len(msg.Key) != 16 {
		t.Errorf("message key length = %d, want 16", len(msg.Key))
	}
	if string(msg.Value) != "test-payload" {
		t.Errorf("message value = %v, want test-payload", string(msg.Value))
	}
	if msg.Time != time.Unix(alert.EventTS, 0) {
		t.Errorf("message time = %v, want %v", msg.Time, time.Unix(alert.EventTS, 0))
	}

	// Check headers
	headerMap := make(map[string]string)
	for _, h := range msg.Headers {
		headerMap[h.Key] = string(h.Value)
	}
	if headerMap["content-type"] != "application/x-protobuf" {
		t.Errorf("content-type header = %v, want application/x-protobuf", headerMap["content-type"])
	}
	if headerMap["schema_version"] != "1" {
		t.Errorf("schema_version header = %v, want 1", headerMap["schema_version"])
	}
	if headerMap["severity"] != "HIGH" {
		t.Errorf("severity header = %v, want HIGH", headerMap["severity"])
	}
}

func TestHashAlertID_Deterministic(t *testing.T) {
	id := "test-alert-id"
	hash1 := hashAlertID(id)
	hash2 := hashAlertID(id)

	if string(hash1) != string(hash2) {
		t.Error("hashAlertID should be deterministic")
	}
}

func TestHashAlertID_DifferentInputs(t *testing.T) {
	hash1 := hashAlertID("id1")
	hash2 := hashAlertID("id2")

	if string(hash1) == string(hash2) {
		t.Error("different inputs should produce different hashes")
	}
}
