# alert-producer – Active Context

## Current Status
Service implementation complete. Ready for testing with Kafka.

## Structure
- `cmd/alert-producer/main.go` - CLI entry point with flags and main loop
- `internal/config/` - Configuration parsing and validation
- `internal/generator/` - Alert generation with weighted distributions
- `internal/producer/` - Kafka producer wrapper

## Recent Changes
- Updated default distributions to match test-data generator values
- Alerts now use the same severity/source/name values as rules in the database
- Boilerplate alert changed to HIGH/api/timeout (common rule combination)

## Next Steps
- Test with local Kafka instance
- Verify message format matches evaluator expectations
- Load testing with various RPS values
- Verify alerts match existing rules and trigger notifications
- UI Integration: Add HTTP API wrapper for rule-service-ui integration

## UI Integration (Planned)

### Overview
The rule-service-ui project will include a new section for generating alerts from alert-producer with optional manual configuration. This allows users to trigger alert generation directly from the web interface instead of using the CLI.

### Integration Approach
- **HTTP API Wrapper**: Create an HTTP server that wraps alert-producer functionality
- **RESTful Endpoints**: Expose alert generation as REST API endpoints
- **Configuration Options**: Support all CLI flags via JSON request body
- **Async Execution**: Run alert generation in background goroutines with job tracking
- **Status Monitoring**: Provide endpoints to check generation status and progress

### UI Features
- **Quick Start**: Pre-configured presets (e.g., "Test Mode", "Burst 100", "Load Test")
- **Manual Configuration**: Full control over all parameters:
  - RPS (alerts per second)
  - Duration or burst size
  - Severity/source/name distributions
  - Seed for deterministic generation
  - Kafka broker/topic settings
- **Real-time Status**: Display generation progress, alerts sent, errors
- **History**: Track previous alert generation runs

### API Design (Planned)
- `POST /api/v1/alerts/generate` - Start alert generation with configuration
- `GET /api/v1/alerts/generate/{jobId}` - Get generation status
- `POST /api/v1/alerts/generate/{jobId}/stop` - Stop running generation
- `GET /api/v1/alerts/generate` - List recent generation jobs

### Implementation Notes
- Reuse existing `internal/processor`, `internal/generator`, and `internal/producer` packages
- Add new `cmd/alert-producer-api` for HTTP server
- Maintain backward compatibility with CLI (`cmd/alert-producer/main.go`)
- Use same configuration validation and error handling
