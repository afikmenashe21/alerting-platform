package events

import (
	"testing"

	pbcommon "github.com/afikmenashe/alerting-platform/pkg/proto/common"
)

func TestSeverityFromProto(t *testing.T) {
	tests := []struct {
		name string
		sev  pbcommon.Severity
		want string
	}{
		{"LOW", pbcommon.Severity_LOW, "LOW"},
		{"MEDIUM", pbcommon.Severity_MEDIUM, "MEDIUM"},
		{"HIGH", pbcommon.Severity_HIGH, "HIGH"},
		{"CRITICAL", pbcommon.Severity_CRITICAL, "CRITICAL"},
		{"UNSPECIFIED", pbcommon.Severity_UNSPECIFIED, "UNSPECIFIED"},
		{"unknown value", pbcommon.Severity(999), "UNSPECIFIED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SeverityFromProto(tt.sev)
			if got != tt.want {
				t.Errorf("SeverityFromProto(%v) = %q, want %q", tt.sev, got, tt.want)
			}
		})
	}
}

func TestSeverityToProto(t *testing.T) {
	tests := []struct {
		name string
		sev  string
		want pbcommon.Severity
	}{
		{"LOW", "LOW", pbcommon.Severity_LOW},
		{"MEDIUM", "MEDIUM", pbcommon.Severity_MEDIUM},
		{"HIGH", "HIGH", pbcommon.Severity_HIGH},
		{"CRITICAL", "CRITICAL", pbcommon.Severity_CRITICAL},
		{"UNSPECIFIED", "UNSPECIFIED", pbcommon.Severity_UNSPECIFIED},
		{"empty string", "", pbcommon.Severity_UNSPECIFIED},
		{"unknown value", "UNKNOWN", pbcommon.Severity_UNSPECIFIED},
		{"lowercase", "low", pbcommon.Severity_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SeverityToProto(tt.sev)
			if got != tt.want {
				t.Errorf("SeverityToProto(%q) = %v, want %v", tt.sev, got, tt.want)
			}
		})
	}
}

func TestSeverityRoundTrip(t *testing.T) {
	// Test that converting from proto to string and back preserves the value
	severities := []pbcommon.Severity{
		pbcommon.Severity_LOW,
		pbcommon.Severity_MEDIUM,
		pbcommon.Severity_HIGH,
		pbcommon.Severity_CRITICAL,
	}

	for _, sev := range severities {
		str := SeverityFromProto(sev)
		got := SeverityToProto(str)
		if got != sev {
			t.Errorf("Round trip failed for %v: got %v after converting to %q", sev, got, str)
		}
	}
}
