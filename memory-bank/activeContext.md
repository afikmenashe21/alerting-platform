# Alerting Platform – Active Context

## What we’re doing right now
We are building the MVP services in Go and want a Cursor “Memory Bank” to capture the exact design decisions and contracts.

Completed:
- ✅ Event contracts + topic names defined
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

## Decisions locked for MVP
- Rules support exact match and wildcard "*" on (severity, source, name).
- Dedupe in aggregator DB unique constraint.
- Redis snapshot warm start (no evaluator DB reads).
- JSON messages with schema_version.
- Evaluator output: one message per client_id (keyed by client_id for tenant locality).
- Wildcard support: "*" matches any value for that field, enabling multiple rules per client to match same alert.
