// Package database provides database operations for clients, rules, and endpoints.
package database

import (
	"time"
)

// Client represents a client record in the database.
type Client struct {
	ClientID  string    `json:"client_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Rule represents a rule record in the database.
type Rule struct {
	RuleID    string    `json:"rule_id"`
	ClientID  string    `json:"client_id"`
	Severity  string    `json:"severity"`
	Source    string    `json:"source"`
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Endpoint represents an endpoint record in the database.
type Endpoint struct {
	EndpointID string    `json:"endpoint_id"`
	RuleID     string    `json:"rule_id"`
	Type       string    `json:"type"` // email, webhook, slack
	Value      string    `json:"value"` // email address, URL, etc.
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Notification represents a notification record in the database.
type Notification struct {
	NotificationID string            `json:"notification_id"`
	ClientID       string            `json:"client_id"`
	AlertID        string            `json:"alert_id"`
	Severity       string            `json:"severity"`
	Source         string            `json:"source"`
	Name           string            `json:"name"`
	Context        map[string]string `json:"context"`
	RuleIDs        []string          `json:"rule_ids"`
	Status         string            `json:"status"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// ClientListResult contains paginated client results.
type ClientListResult struct {
	Clients []*Client `json:"clients"`
	Total   int64     `json:"total"`
	Limit   int       `json:"limit"`
	Offset  int       `json:"offset"`
}

// RuleListResult contains paginated rule results.
type RuleListResult struct {
	Rules  []*Rule `json:"rules"`
	Total  int64   `json:"total"`
	Limit  int     `json:"limit"`
	Offset int     `json:"offset"`
}

// EndpointListResult contains paginated endpoint results.
type EndpointListResult struct {
	Endpoints []*Endpoint `json:"endpoints"`
	Total     int64       `json:"total"`
	Limit     int         `json:"limit"`
	Offset    int         `json:"offset"`
}
