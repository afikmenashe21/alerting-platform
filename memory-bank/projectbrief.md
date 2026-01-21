# Alerting Platform – Project Brief

## What we’re building
A multi-service Go application that implements an end‑to‑end **alert notification platform**:

- **Upstream** produces alerts with fields: `severity`, `source`, `name` (+ optional context).
- Customers (tenants/clients) define **rules** like: *if alert has severity=HIGH and source=X and name=Y then notify endpoints (email)*.
- Platform evaluates alerts, **dedupes** notifications so the same alert is not sent twice to the same client, persists notification intents, and sends emails (stub) reliably under **at‑least‑once** delivery.

This repo exists to practice **system design + real implementation**: Kafka consumer groups, rule indexing in memory, Redis warm-start snapshots, Postgres idempotency boundary, and clean failure handling.

## High-level flow (topics + services)
1. **alert-producer** → Kafka `alerts.new`
2. **rule-service** (CRUD) → Postgres + Kafka `rule.changed`
3. **rule-updater** consumes `rule.changed` → rebuilds Redis rule snapshot + increments version
4. **evaluator** consumes `alerts.new` → matches rules using in-memory indexes (warm from Redis) → emits `alerts.matched`
5. **aggregator** consumes `alerts.matched` → idempotent insert into `notifications` (dedupe boundary) → emits `notifications.ready`
6. **sender** consumes `notifications.ready` → reads `notifications` row → sends email stub → updates status to `SENT`

## Core correctness requirements
- **At-least-once** processing from Kafka everywhere.
- **No duplicates per (client_id, alert_id)**:
  - Enforced in aggregator DB with **unique constraint**.
- **Rule changes propagate** to evaluator without full DB scan on every restart:
  - Rule-updater builds **Redis snapshot**; evaluator warmups quickly.
- **Multiple rules can match same client** for the same alert:
  - Evaluator produces **one message per client_id**, each containing the alert + all matching rule_ids for that client.
  - If an alert matches multiple clients, multiple messages are published (one per client).
  - Messages are keyed by `client_id` for tenant locality.
  - Aggregator dedupes at notification level (one notification per client+alert), but keeps matched rule_ids for explainability.

## Rule model (MVP)
For MVP rules support exact match and wildcards:
- `severity` (enum: LOW/MEDIUM/HIGH/CRITICAL or "*" for any)
- `source` (string or "*" for any)
- `name` (string or "*" for any)

Wildcard "*" matches any value for that field. At least one field must be non-wildcard.

## Non-goals for MVP
- Baseline/anomaly upstream analytics (separate system)
- Multi-channel delivery (Slack/webhooks) beyond email stub
- Full retry scheduler / DLQ / rate limiting (can add later)
- External schema registry — Kafka events use protobuf messages with a `schema_version` field (see `proto/` + `pkg/proto/`), but no external registry service is used yet.
