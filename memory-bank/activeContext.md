# Alerting Platform – Active Context

## What we’re doing right now
We are building the MVP services in Go and want a Cursor “Memory Bank” to capture the exact design decisions and contracts.

Completed:
- ✅ Event contracts + topic names defined (protobuf messages)
- ✅ alert-producer: generate alerts + load tests
- ✅ evaluator: warmup + matching + `alerts.matched` output (one message per client_id)
- ✅ aggregator: idempotent insert + `notifications.ready` output

In progress / Next:
1) ✅ Postgres migrations for clients/rules (rule-service)
2) ✅ rule-service: CRUD + publish `rule.changed` events
3) ✅ rule-updater: snapshot writer to Redis (consumes rule.changed, rebuilds snapshot, increments version)
4) ✅ sender: consume notifications.ready + send via email (SMTP), Slack (webhook API), and webhook (HTTP POST) + update status
5) ✅ rule-service-ui: React UI for managing clients, rules, and endpoints
6) ✅ Centralized infrastructure management (Postgres, Kafka, Redis, Zookeeper)
7) 🔄 UI Integration for alert-producer: Add HTTP API wrapper and UI component for generating alerts with optional manual config
8) ✅ Protobuf Integration: All Kafka topics now use protobuf messages defined in `proto/*.proto` with generated Go types in `pkg/proto/`; JSON wire format for Kafka events has been fully removed.
9) ✅ Protobuf Enhanced Tooling: Added buf linting, breaking change detection, code verification, CI/CD integration, and pre-commit hooks for robust schema management. CI buf detection fixed (2026-01-21).
10) ✅ Protobuf Severity Alignment: Changed protobuf enum values to match database format directly (LOW, MEDIUM, HIGH, CRITICAL - removed SEVERITY_ prefix) for simpler, cleaner code (2026-01-21).

## Code health
- Completed comprehensive cleanup and modularization across all services:
  - Extracted redundant code patterns (validation, error handling, database scanning)
  - Split large files (>200 lines) by resource/concern where appropriate
  - All services maintain existing functionality with improved organization
  - Remaining files slightly over 200 lines are well-organized handler files without obvious redundancy

## Decisions locked for MVP
- Rules support exact match and wildcard "*" on (severity, source, name).
- Dedupe in aggregator DB unique constraint.
- Redis snapshot warm start (no evaluator DB reads).
- Protobuf messages for all Kafka topics (no JSON wire format on Kafka).
- Evaluator output: one message per client_id (keyed by client_id for tenant locality).
- Wildcard support: "*" matches any value for that field, enabling multiple rules per client to match same alert.
