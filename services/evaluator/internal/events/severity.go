package events

import (
	pbcommon "github.com/afikmenashe/alerting-platform/pkg/proto/common"
)

// SeverityFromProto converts a protobuf Severity enum to its string representation.
// Unknown values are converted to "UNSPECIFIED".
func SeverityFromProto(sev pbcommon.Severity) string {
	switch sev {
	case pbcommon.Severity_LOW:
		return "LOW"
	case pbcommon.Severity_MEDIUM:
		return "MEDIUM"
	case pbcommon.Severity_HIGH:
		return "HIGH"
	case pbcommon.Severity_CRITICAL:
		return "CRITICAL"
	default:
		return "UNSPECIFIED"
	}
}

// SeverityToProto converts a severity string to its protobuf enum representation.
// Unknown values are converted to Severity_UNSPECIFIED.
func SeverityToProto(sev string) pbcommon.Severity {
	switch sev {
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
