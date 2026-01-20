// Package database provides database operations for notifications and endpoints tables.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// Endpoint represents an endpoint record from the endpoints table.
type Endpoint struct {
	EndpointID string
	RuleID     string
	Type       string
	Value      string
	Enabled    bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// GetEmailEndpointsByRuleIDs retrieves all enabled email endpoints for the given rule IDs.
// Returns a map of rule_id -> []email addresses.
// Note: ruleIDs may be UUIDs (from actual rules) or test strings (like "rule-001").
// The query casts rule_id to text for comparison, so it works with both.
// DEPRECATED: Use GetEndpointsByRuleIDs instead to get all endpoint types.
func (db *DB) GetEmailEndpointsByRuleIDs(ctx context.Context, ruleIDs []string) (map[string][]string, error) {
	endpoints, err := db.GetEndpointsByRuleIDs(ctx, ruleIDs)
	if err != nil {
		return nil, err
	}

	// Filter to only email endpoints
	result := make(map[string][]string)
	for ruleID, eps := range endpoints {
		for _, ep := range eps {
			if ep.Type == "email" {
				result[ruleID] = append(result[ruleID], ep.Value)
			}
		}
	}
	return result, nil
}

// GetEndpointsByRuleIDs retrieves all enabled endpoints (email, slack, webhook) for the given rule IDs.
// Returns a map of rule_id -> []Endpoint.
// Note: ruleIDs may be UUIDs (from actual rules) or test strings (like "rule-001").
// The query casts rule_id to text for comparison, so it works with both.
func (db *DB) GetEndpointsByRuleIDs(ctx context.Context, ruleIDs []string) (map[string][]Endpoint, error) {
	if len(ruleIDs) == 0 {
		return make(map[string][]Endpoint), nil
	}

	// Filter out empty rule IDs and convert to array for PostgreSQL
	validRuleIDs := make([]string, 0, len(ruleIDs))
	for _, id := range ruleIDs {
		if id != "" {
			validRuleIDs = append(validRuleIDs, id)
		}
	}

	if len(validRuleIDs) == 0 {
		return make(map[string][]Endpoint), nil
	}

	// Cast rule_id::text to compare with TEXT[] elements
	// This handles both UUID rule_ids and test string rule_ids
	query := `
		SELECT rule_id::text, type, value, endpoint_id, enabled, created_at, updated_at
		FROM endpoints
		WHERE rule_id::text = ANY($1) AND enabled = TRUE
		ORDER BY rule_id, created_at ASC
	`

	rows, err := db.conn.QueryContext(ctx, query, pq.Array(validRuleIDs))
	if err != nil {
		return nil, fmt.Errorf("failed to query endpoints: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]Endpoint)
	for rows.Next() {
		var ep Endpoint
		if err := rows.Scan(&ep.RuleID, &ep.Type, &ep.Value, &ep.EndpointID, &ep.Enabled, &ep.CreatedAt, &ep.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan endpoint: %w", err)
		}
		result[ep.RuleID] = append(result[ep.RuleID], ep)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating endpoints: %w", err)
	}

	return result, nil
}
