// Package provider defines the email provider interface and registry.
// It uses the Strategy pattern to support multiple email backends (SES, Resend, etc.)
package provider

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
)

// EmailRequest represents an email to be sent.
type EmailRequest struct {
	From    string
	To      []string
	Subject string
	Body    string   // Plain text body
	HTML    string   // HTML body (optional)
}

// Provider is the interface that all email providers must implement.
type Provider interface {
	// Name returns the provider name (e.g., "ses", "resend")
	Name() string

	// Send sends an email using this provider.
	Send(ctx context.Context, req *EmailRequest) error

	// IsConfigured returns true if the provider is properly configured.
	IsConfigured() bool
}

// Registry manages email providers with fallback support.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
	primary   string   // Primary provider name
	fallback  []string // Fallback provider names in order
}

// NewRegistry creates a new email provider registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
		fallback:  make([]string, 0),
	}
}

// Register adds a provider to the registry.
func (r *Registry) Register(provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[provider.Name()] = provider
	slog.Info("Registered email provider", "name", provider.Name(), "configured", provider.IsConfigured())
}

// SetPrimary sets the primary provider by name.
func (r *Registry) SetPrimary(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.providers[name]; !ok {
		return fmt.Errorf("provider %q not registered", name)
	}
	r.primary = name
	slog.Info("Set primary email provider", "name", name)
	return nil
}

// SetFallback sets the fallback providers in order.
func (r *Registry) SetFallback(names ...string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, name := range names {
		if _, ok := r.providers[name]; !ok {
			return fmt.Errorf("provider %q not registered", name)
		}
	}
	r.fallback = names
	slog.Info("Set fallback email providers", "order", names)
	return nil
}

// Get returns a provider by name.
func (r *Registry) Get(name string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	return p, ok
}

// GetPrimary returns the primary configured provider.
// Falls back to other providers if primary is not configured.
func (r *Registry) GetPrimary() (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Try primary first
	if r.primary != "" {
		if p, ok := r.providers[r.primary]; ok && p.IsConfigured() {
			return p, nil
		}
	}

	// Try fallbacks in order
	for _, name := range r.fallback {
		if p, ok := r.providers[name]; ok && p.IsConfigured() {
			slog.Warn("Primary email provider not configured, using fallback",
				"primary", r.primary,
				"fallback", name,
			)
			return p, nil
		}
	}

	// Try any configured provider
	for name, p := range r.providers {
		if p.IsConfigured() {
			slog.Warn("Using first available email provider", "name", name)
			return p, nil
		}
	}

	return nil, fmt.Errorf("no configured email provider available")
}

// Send sends an email using the best available provider.
func (r *Registry) Send(ctx context.Context, req *EmailRequest) error {
	provider, err := r.GetPrimary()
	if err != nil {
		return err
	}

	err = provider.Send(ctx, req)
	if err != nil {
		// Try fallback providers on failure
		r.mu.RLock()
		fallbacks := r.fallback
		r.mu.RUnlock()

		for _, name := range fallbacks {
			p, ok := r.Get(name)
			if !ok || !p.IsConfigured() || p.Name() == provider.Name() {
				continue
			}

			slog.Warn("Primary provider failed, trying fallback",
				"primary", provider.Name(),
				"fallback", name,
				"error", err,
			)

			if fallbackErr := p.Send(ctx, req); fallbackErr == nil {
				return nil // Success with fallback
			}
		}
		return err // Return original error
	}

	return nil
}

// List returns all registered provider names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// GetEnvOrDefault returns env var value or default.
func GetEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}
