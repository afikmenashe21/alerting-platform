// Package handlers provides HTTP handlers for the rule-service API.
package handlers

import (
	"rule-service/internal/database"
	"rule-service/internal/producer"
)

const (
	SchemaVersion = 1
)

// Handlers wraps dependencies for HTTP handlers.
type Handlers struct {
	db       *database.DB
	producer *producer.Producer
}

// NewHandlers creates a new handlers instance.
func NewHandlers(db *database.DB, producer *producer.Producer) *Handlers {
	return &Handlers{
		db:       db,
		producer: producer,
	}
}
