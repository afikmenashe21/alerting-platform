// Package kafka provides shared Kafka utilities for the evaluator service.
package kafka

import "time"

const (
	// ReadTimeout is the maximum time to wait for a Kafka read operation.
	ReadTimeout = 10 * time.Second
	// CommitInterval is how often to commit offsets (after processing).
	CommitInterval = 1 * time.Second
	// WriteTimeout is the maximum time to wait for a Kafka write operation.
	WriteTimeout = 10 * time.Second
)
