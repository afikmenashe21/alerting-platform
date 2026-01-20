// Package api provides HTTP API handlers and job management for alert-producer.
package api

import (
	"fmt"
	"time"

	"alert-producer/internal/config"
)

// GenerateRequest represents a request to generate alerts.
type GenerateRequest struct {
	RPS          *float64 `json:"rps,omitempty"`
	Duration     string   `json:"duration,omitempty"` // e.g., "60s", "5m"
	BurstSize    *int     `json:"burst,omitempty"`
	Seed         *int64   `json:"seed,omitempty"`
	SeverityDist string   `json:"severity_dist,omitempty"`
	SourceDist   string   `json:"source_dist,omitempty"`
	NameDist     string   `json:"name_dist,omitempty"`
	KafkaBrokers string   `json:"kafka_brokers,omitempty"`
	Topic        string   `json:"topic,omitempty"`
	Mock         bool     `json:"mock,omitempty"`
	Test         bool     `json:"test,omitempty"`
	SingleTest   bool     `json:"single_test,omitempty"`
	// Single alert properties (used when single_test is true)
	Severity     string   `json:"severity,omitempty"` // e.g., "HIGH", "LOW", "MEDIUM", "CRITICAL"
	Source       string   `json:"source,omitempty"`   // e.g., "api", "db", "cache"
	Name         string   `json:"name,omitempty"`     // e.g., "timeout", "error", "crash"
}

// ToConfig converts a GenerateRequest to a config.Config.
func (req *GenerateRequest) ToConfig(defaultKafkaBrokers string) (config.Config, error) {
	cfg := config.Config{
		KafkaBrokers: defaultKafkaBrokers,
		Topic:        "alerts.new",
		RPS:          10.0,
		Duration:     60 * time.Second,
		BurstSize:    0,
		Seed:         0,
		SeverityDist: "HIGH:30,MEDIUM:30,LOW:25,CRITICAL:15",
		SourceDist:   "api:25,db:20,cache:15,monitor:15,queue:10,worker:5,frontend:5,backend:5",
		NameDist:     "timeout:15,error:15,crash:10,slow:10,memory:10,cpu:10,disk:10,network:10,auth:5,validation:5",
	}

	// Override with request values
	if req.KafkaBrokers != "" {
		cfg.KafkaBrokers = req.KafkaBrokers
	}
	if req.Topic != "" {
		cfg.Topic = req.Topic
	}
	if req.RPS != nil {
		cfg.RPS = *req.RPS
	}
	if req.Duration != "" {
		duration, err := time.ParseDuration(req.Duration)
		if err != nil {
			return cfg, fmt.Errorf("invalid duration format: %w", err)
		}
		cfg.Duration = duration
	}
	if req.BurstSize != nil {
		cfg.BurstSize = *req.BurstSize
	}
	if req.Seed != nil {
		cfg.Seed = *req.Seed
	}
	if req.SeverityDist != "" {
		cfg.SeverityDist = req.SeverityDist
	}
	if req.SourceDist != "" {
		cfg.SourceDist = req.SourceDist
	}
	if req.NameDist != "" {
		cfg.NameDist = req.NameDist
	}

	return cfg, nil
}

// GenerateResponse represents the response to a generate request.
type GenerateResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

// JobResponse represents a job status response.
type JobResponse struct {
	ID          string    `json:"id"`
	Status      string    `json:"status"`
	Config      *GenerateRequest `json:"config"`
	CreatedAt   time.Time `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	AlertsSent  int64     `json:"alerts_sent"`
	Error       string    `json:"error,omitempty"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error string `json:"error"`
}
