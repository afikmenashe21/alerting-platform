// Package reloader handles polling Redis for rule version changes and hot-reloading indexes.
// It also supports consuming rule.changed events from Kafka for immediate updates.
package reloader

import (
	"context"
	"log/slog"
	"time"

	"evaluator/internal/indexes"
	"evaluator/internal/matcher"
	"evaluator/internal/snapshot"
)

// Reloader polls Redis for version changes and reloads rule indexes when needed.
// It can also consume rule.changed events from Kafka for immediate updates.
type Reloader struct {
	loader         *snapshot.Loader
	matcher        *matcher.Matcher
	pollInterval   time.Duration
	currentVersion int64
}

// NewReloader creates a new reloader with the given dependencies.
func NewReloader(loader *snapshot.Loader, matcher *matcher.Matcher, pollInterval time.Duration) *Reloader {
	return &Reloader{
		loader:      loader,
		matcher:     matcher,
		pollInterval: pollInterval,
	}
}

// Start begins polling Redis for version changes in a background goroutine.
// It will reload indexes atomically when the version changes.
// The goroutine will exit when ctx is cancelled.
func (r *Reloader) Start(ctx context.Context) error {
	// Get initial version
	version, err := r.loader.GetVersion(ctx)
	if err != nil {
		return err
	}
	r.currentVersion = version

	slog.Info("Starting version poller",
		"poll_interval", r.pollInterval,
		"initial_version", r.currentVersion,
	)

	go r.pollLoop(ctx)
	return nil
}

// pollLoop continuously polls Redis for version changes.
func (r *Reloader) pollLoop(ctx context.Context) {
	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Version poller stopped")
			return
		case <-ticker.C:
			if err := r.checkAndReload(ctx); err != nil {
				slog.Error("Failed to check/reload rules",
					"error", err,
				)
				// Continue polling even if reload fails
			}
		}
	}
}

// checkAndReload checks if the version has changed and reloads if needed.
func (r *Reloader) checkAndReload(ctx context.Context) error {
	version, err := r.loader.GetVersion(ctx)
	if err != nil {
		return err
	}

	if version == r.currentVersion {
		return nil // No change
	}

	slog.Info("Rule version changed, reloading indexes",
		"old_version", r.currentVersion,
		"new_version", version,
	)

	// Load new snapshot
	snap, err := r.loader.LoadSnapshot(ctx)
	if err != nil {
		return err
	}

	// Build new indexes
	newIndexes := indexes.NewIndexes(snap)

	// Atomically swap indexes
	r.matcher.UpdateIndexes(newIndexes)
	r.currentVersion = version

	slog.Info("Indexes reloaded successfully",
		"version", version,
		"rules_count", newIndexes.RuleCount(),
	)

	return nil
}

// ReloadNow forces an immediate reload of indexes from Redis snapshot.
// This can be called when a rule.changed event is received.
func (r *Reloader) ReloadNow(ctx context.Context) error {
	return r.checkAndReload(ctx)
}
