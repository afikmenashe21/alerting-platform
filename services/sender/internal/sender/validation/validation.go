// Package validation provides shared validation utilities for sender implementations.
package validation

import "strings"

// IsValidURL checks if a string is a valid HTTP/HTTPS URL.
func IsValidURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
