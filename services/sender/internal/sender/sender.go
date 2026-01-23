// Package sender provides a coordinator for multi-channel notification sending.
// It uses the strategy pattern to route notifications to appropriate senders.
package sender

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"sender/internal/database"
	"sender/internal/sender/email"
	"sender/internal/sender/retry"
	"sender/internal/sender/slack"
	"sender/internal/sender/strategy"
	"sender/internal/sender/webhook"
)

// Sender coordinates notification sending across multiple channels.
type Sender struct {
	registry *strategy.Registry
}

// NewSender creates a new sender coordinator with all strategies registered.
func NewSender() *Sender {
	registry := strategy.NewRegistry()

	// Register all sender strategies
	registry.Register(email.NewSender())
	registry.Register(slack.NewSender())
	registry.Register(webhook.NewSender())

	return &Sender{
		registry: registry,
	}
}

// NewSenderWithRegistry creates a new sender coordinator with a custom registry.
// This is useful for testing or custom sender configurations.
func NewSenderWithRegistry(registry *strategy.Registry) *Sender {
	return &Sender{
		registry: registry,
	}
}

// SendNotification sends notifications to all relevant endpoints for the given notification.
// It supports email, Slack, and webhook endpoints using the strategy pattern.
func (s *Sender) SendNotification(ctx context.Context, notification *database.Notification, endpoints map[string][]database.Endpoint) error {
	if len(endpoints) == 0 {
		slog.Warn("No endpoints found for notification",
			"notification_id", notification.NotificationID,
			"rule_ids", notification.RuleIDs,
		)
		return fmt.Errorf("no endpoints found for notification %s", notification.NotificationID)
	}

	// Group endpoints by type and value
	endpointsByType := s.groupEndpoints(endpoints, notification.RuleIDs)

	// Send to all endpoint types
	var errors []string
	totalEndpoints := 0
	successfulSends := 0

	for endpointType, endpointValues := range endpointsByType {
		sender, ok := s.registry.Get(endpointType)
		if !ok {
			slog.Warn("Unknown endpoint type, skipping",
				"type", endpointType,
				"notification_id", notification.NotificationID,
			)
			continue
		}

		totalEndpoints += len(endpointValues)
		for _, endpointValue := range endpointValues {
			// Use retry with exponential backoff for transient failures
			retryCfg := retry.DefaultConfig()
			operation := fmt.Sprintf("send_%s_%s", endpointType, notification.NotificationID)

			err := retry.WithRetry(ctx, retryCfg, operation, func() error {
				return sender.Send(ctx, endpointValue, notification)
			})

			if err != nil {
				errors = append(errors, fmt.Sprintf("%s (%s): %s", endpointType, endpointValue, err.Error()))
			} else {
				successfulSends++
			}
		}
	}

	// If all sends failed, return error
	if len(errors) > 0 && successfulSends == 0 {
		return fmt.Errorf("all sends failed: %s", strings.Join(errors, "; "))
	}

	// If some sends failed, log warning but don't fail
	if len(errors) > 0 {
		slog.Warn("Some sends failed",
			"notification_id", notification.NotificationID,
			"successful", successfulSends,
			"failed", len(errors),
			"errors", strings.Join(errors, "; "),
		)
	}

	return nil
}

// groupEndpoints groups endpoints by type and collects unique values.
// Returns a map of endpoint type -> slice of unique endpoint values.
func (s *Sender) groupEndpoints(endpoints map[string][]database.Endpoint, ruleIDs []string) map[string][]string {
	// Use a map to track unique values per type
	valueSet := make(map[string]map[string]bool)

	for _, ruleID := range ruleIDs {
		if eps, ok := endpoints[ruleID]; ok {
			for _, ep := range eps {
				if ep.Enabled {
					if valueSet[ep.Type] == nil {
						valueSet[ep.Type] = make(map[string]bool)
					}
					valueSet[ep.Type][ep.Value] = true
				}
			}
		}
	}

	// Convert sets to slices
	result := make(map[string][]string)
	for endpointType, values := range valueSet {
		valueSlice := make([]string, 0, len(values))
		for value := range values {
			valueSlice = append(valueSlice, value)
		}
		result[endpointType] = valueSlice
	}

	return result
}
