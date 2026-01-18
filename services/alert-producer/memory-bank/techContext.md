# alert-producer – Tech Context

- Go 1.22+
- Kafka producer: alerts.new
- Config via flags/env
- **Planned**: HTTP API server for UI integration (rule-service-ui)
  - RESTful endpoints for alert generation
  - Job tracking and status monitoring
  - Same core logic as CLI, exposed via HTTP
