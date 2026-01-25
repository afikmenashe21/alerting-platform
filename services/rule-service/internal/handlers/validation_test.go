// Package handlers provides tests for validation and HTTP helpers.
package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParsePagination(t *testing.T) {
	tests := []struct {
		name           string
		queryString    string
		expectedLimit  int
		expectedOffset int
	}{
		{
			name:           "defaults when no params",
			queryString:    "",
			expectedLimit:  50,
			expectedOffset: 0,
		},
		{
			name:           "custom limit",
			queryString:    "limit=100",
			expectedLimit:  100,
			expectedOffset: 0,
		},
		{
			name:           "custom offset",
			queryString:    "offset=25",
			expectedLimit:  50,
			expectedOffset: 25,
		},
		{
			name:           "both custom",
			queryString:    "limit=200&offset=50",
			expectedLimit:  200,
			expectedOffset: 50,
		},
		{
			name:           "invalid limit uses default",
			queryString:    "limit=invalid",
			expectedLimit:  50,
			expectedOffset: 0,
		},
		{
			name:           "negative limit uses default",
			queryString:    "limit=-5",
			expectedLimit:  50,
			expectedOffset: 0,
		},
		{
			name:           "zero limit uses default",
			queryString:    "limit=0",
			expectedLimit:  50,
			expectedOffset: 0,
		},
		{
			name:           "negative offset uses default",
			queryString:    "offset=-10",
			expectedLimit:  50,
			expectedOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test?"+tt.queryString, nil)
			p := parsePagination(req)

			if p.Limit != tt.expectedLimit {
				t.Errorf("Limit = %d, want %d", p.Limit, tt.expectedLimit)
			}
			if p.Offset != tt.expectedOffset {
				t.Errorf("Offset = %d, want %d", p.Offset, tt.expectedOffset)
			}
		})
	}
}

func TestIsValidSeverity(t *testing.T) {
	validCases := []string{"LOW", "MEDIUM", "HIGH", "CRITICAL", "*"}
	invalidCases := []string{"low", "INVALID", "", "medium"}

	for _, s := range validCases {
		if !isValidSeverity(s) {
			t.Errorf("isValidSeverity(%q) = false, want true", s)
		}
	}

	for _, s := range invalidCases {
		if isValidSeverity(s) {
			t.Errorf("isValidSeverity(%q) = true, want false", s)
		}
	}
}

func TestIsValidEndpointType(t *testing.T) {
	validCases := []string{"email", "webhook", "slack"}
	invalidCases := []string{"sms", "EMAIL", "", "unknown"}

	for _, s := range validCases {
		if !isValidEndpointType(s) {
			t.Errorf("isValidEndpointType(%q) = false, want true", s)
		}
	}

	for _, s := range invalidCases {
		if isValidEndpointType(s) {
			t.Errorf("isValidEndpointType(%q) = true, want false", s)
		}
	}
}

func TestIsAllWildcards(t *testing.T) {
	if !isAllWildcards("*", "*", "*") {
		t.Error("isAllWildcards(*, *, *) = false, want true")
	}
	if isAllWildcards("HIGH", "*", "*") {
		t.Error("isAllWildcards(HIGH, *, *) = true, want false")
	}
	if isAllWildcards("*", "source", "*") {
		t.Error("isAllWildcards(*, source, *) = true, want false")
	}
}
