// Package api provides HTTP API handlers and job management for alert-producer.
// This file is kept for backward compatibility but handlers have been split into:
// - types.go: Request/Response types
// - generate.go: HandleGenerate
// - job_handlers.go: HandleGetJob, HandleListJobs, HandleStopJob
// - health.go: HandleHealth
// - helpers.go: Response helpers and validation
package api

