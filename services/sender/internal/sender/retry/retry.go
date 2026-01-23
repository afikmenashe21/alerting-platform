// Package retry provides retry logic with exponential backoff for transient failures.
package retry

import (
	"context"
	"log/slog"
	"math"
	"math/rand"
	"strings"
	"time"
)

// Config defines retry behavior.
type Config struct {
	MaxRetries     int           // Maximum number of retry attempts (0 = no retries)
	InitialBackoff time.Duration // Initial backoff duration
	MaxBackoff     time.Duration // Maximum backoff duration
	BackoffFactor  float64       // Multiplier for exponential backoff
}

// DefaultConfig returns sensible default retry configuration.
func DefaultConfig() Config {
	return Config{
		MaxRetries:     3,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     5 * time.Second,
		BackoffFactor:  2.0,
	}
}

// IsRetryable checks if an error is retryable (transient).
// Network errors, rate limits, and temporary service unavailability are retryable.
// Validation errors and permanent failures are not.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Non-retryable errors (permanent failures)
	nonRetryable := []string{
		"not verified",           // SES sandbox - recipient not verified
		"validation error",       // Invalid input
		"invalid",                // Invalid request
		"malformed",              // Bad request format
		"email address is empty", // Missing required field
		"recipient is required",  // Missing required field
	}

	for _, s := range nonRetryable {
		if strings.Contains(errStr, s) {
			return false
		}
	}

	// Retryable errors (transient failures)
	retryable := []string{
		"timeout",           // Network timeout
		"connection refused", // Service temporarily unavailable
		"connection reset",   // Network hiccup
		"temporary",          // Explicit temporary error
		"rate limit",         // Rate limiting
		"throttl",            // Throttling
		"503",                // Service unavailable
		"502",                // Bad gateway
		"504",                // Gateway timeout
		"too many requests",  // Rate limiting
		"try again",          // Server suggests retry
	}

	for _, s := range retryable {
		if strings.Contains(errStr, s) {
			return true
		}
	}

	// Default: don't retry unknown errors
	return false
}

// WithRetry executes a function with retry logic and exponential backoff.
// It only retries on transient errors determined by IsRetryable.
func WithRetry(ctx context.Context, cfg Config, operation string, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		// Execute the operation
		err := fn()
		if err == nil {
			if attempt > 0 {
				slog.Info("Operation succeeded after retry",
					"operation", operation,
					"attempt", attempt+1,
				)
			}
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !IsRetryable(err) {
			slog.Debug("Error is not retryable, failing immediately",
				"operation", operation,
				"error", err,
			)
			return err
		}

		// Check if we have retries left
		if attempt >= cfg.MaxRetries {
			slog.Warn("Max retries exceeded",
				"operation", operation,
				"attempts", attempt+1,
				"error", err,
			)
			return err
		}

		// Calculate backoff with jitter
		backoff := calculateBackoff(cfg, attempt)

		slog.Warn("Operation failed, retrying",
			"operation", operation,
			"attempt", attempt+1,
			"max_attempts", cfg.MaxRetries+1,
			"backoff", backoff,
			"error", err,
		)

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			// Continue to next attempt
		}
	}

	return lastErr
}

// calculateBackoff calculates the backoff duration with jitter.
func calculateBackoff(cfg Config, attempt int) time.Duration {
	// Exponential backoff: initial * factor^attempt
	backoff := float64(cfg.InitialBackoff) * math.Pow(cfg.BackoffFactor, float64(attempt))

	// Cap at max backoff
	if backoff > float64(cfg.MaxBackoff) {
		backoff = float64(cfg.MaxBackoff)
	}

	// Add jitter (±25%)
	jitter := backoff * 0.25 * (rand.Float64()*2 - 1)
	backoff += jitter

	return time.Duration(backoff)
}
