# rule-service – Active Context

## Completed
- ✅ Database schema: clients, rules, and endpoints tables with proper foreign keys
- ✅ CRUD API: HTTP REST endpoints for clients, rules, and endpoints
- ✅ Kafka producer: publishes rule.changed events after DB commits
- ✅ Optimistic locking: version-based updates for rules
- ✅ Event actions: CREATED, UPDATED, DELETED, DISABLED
- ✅ Code cleanup and modularization:
  - Removed redundant code via private helpers (validation, HTTP, JSON parsing)
  - Modularized handlers package: split 674-line file into resource-specific files:
    - `clients.go`, `rules.go`, `endpoints.go`, `notifications.go`
  - Added `publishRuleChangedEvent()` helper to reduce duplication in rule handlers
  - All tests pass; behavior unchanged

## Database Schema
- **clients**: client_id (PK), name, timestamps
- **rules**: rule_id (PK), client_id (FK → clients), severity, source, name, enabled, version, timestamps
- **endpoints**: endpoint_id (PK), rule_id (FK → rules, CASCADE), type (email/webhook/slack), value, enabled, timestamps
- Relationships: Client → Rules (1:N), Rule → Endpoints (1:N)
- Unique constraints: (client_id, severity, source, name) for rules, (rule_id, type, value) for endpoints

## Implementation Details
- Rules use exact-match on (severity, source, name)
- Endpoints support multiple types: email, webhook, slack
- Email fields removed from clients and rules (now managed via endpoints table)
- Events keyed by rule_id for Kafka partitioning
- HTTP server on port 8081 (configurable)
- Graceful shutdown handling
