package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "timeout error",
			err:      errors.New("connection timeout"),
			expected: true,
		},
		{
			name:     "rate limit error",
			err:      errors.New("rate limit exceeded"),
			expected: true,
		},
		{
			name:     "503 service unavailable",
			err:      errors.New("503 Service Unavailable"),
			expected: true,
		},
		{
			name:     "connection refused",
			err:      errors.New("dial tcp: connection refused"),
			expected: true,
		},
		{
			name:     "SES not verified (permanent)",
			err:      errors.New("Email address is not verified"),
			expected: false,
		},
		{
			name:     "validation error (permanent)",
			err:      errors.New("validation error: invalid email"),
			expected: false,
		},
		{
			name:     "generic error",
			err:      errors.New("some random error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetryable(tt.err)
			if got != tt.expected {
				t.Errorf("IsRetryable(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestWithRetry_Success(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		MaxRetries:     3,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
		BackoffFactor:  2.0,
	}

	callCount := 0
	err := WithRetry(ctx, cfg, "test", func() error {
		callCount++
		return nil
	})

	if err != nil {
		t.Errorf("WithRetry() error = %v, want nil", err)
	}
	if callCount != 1 {
		t.Errorf("WithRetry() called function %d times, want 1", callCount)
	}
}

func TestWithRetry_RetryableError(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		MaxRetries:     2,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
		BackoffFactor:  2.0,
	}

	callCount := 0
	err := WithRetry(ctx, cfg, "test", func() error {
		callCount++
		if callCount < 3 {
			return errors.New("connection timeout")
		}
		return nil
	})

	if err != nil {
		t.Errorf("WithRetry() error = %v, want nil", err)
	}
	if callCount != 3 {
		t.Errorf("WithRetry() called function %d times, want 3", callCount)
	}
}

func TestWithRetry_NonRetryableError(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		MaxRetries:     3,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
		BackoffFactor:  2.0,
	}

	callCount := 0
	expectedErr := errors.New("Email address is not verified")
	err := WithRetry(ctx, cfg, "test", func() error {
		callCount++
		return expectedErr
	})

	if err != expectedErr {
		t.Errorf("WithRetry() error = %v, want %v", err, expectedErr)
	}
	if callCount != 1 {
		t.Errorf("WithRetry() called function %d times, want 1 (no retries for non-retryable)", callCount)
	}
}

func TestWithRetry_MaxRetriesExceeded(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		MaxRetries:     2,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
		BackoffFactor:  2.0,
	}

	callCount := 0
	expectedErr := errors.New("connection timeout")
	err := WithRetry(ctx, cfg, "test", func() error {
		callCount++
		return expectedErr
	})

	if err != expectedErr {
		t.Errorf("WithRetry() error = %v, want %v", err, expectedErr)
	}
	if callCount != 3 { // 1 initial + 2 retries
		t.Errorf("WithRetry() called function %d times, want 3", callCount)
	}
}

func TestWithRetry_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := Config{
		MaxRetries:     10,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     time.Second,
		BackoffFactor:  2.0,
	}

	callCount := 0
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := WithRetry(ctx, cfg, "test", func() error {
		callCount++
		return errors.New("connection timeout")
	})

	if err != context.Canceled {
		t.Errorf("WithRetry() error = %v, want context.Canceled", err)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxRetries != 3 {
		t.Errorf("DefaultConfig().MaxRetries = %d, want 3", cfg.MaxRetries)
	}
	if cfg.InitialBackoff != 100*time.Millisecond {
		t.Errorf("DefaultConfig().InitialBackoff = %v, want 100ms", cfg.InitialBackoff)
	}
	if cfg.MaxBackoff != 5*time.Second {
		t.Errorf("DefaultConfig().MaxBackoff = %v, want 5s", cfg.MaxBackoff)
	}
	if cfg.BackoffFactor != 2.0 {
		t.Errorf("DefaultConfig().BackoffFactor = %f, want 2.0", cfg.BackoffFactor)
	}
}
