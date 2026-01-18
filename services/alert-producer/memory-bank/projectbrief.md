# alert-producer – Project Brief

## Purpose
Generate synthetic alerts and publish them to Kafka `alerts.new` for testing the whole pipeline.

## Features
- Configurable RPS and duration
- Severity/source/name distributions
- Deterministic seed mode (optional)
- **Planned**: HTTP API for UI integration (rule-service-ui)
  - Web interface for alert generation
  - Optional manual configuration of all parameters
  - Real-time status and progress monitoring
