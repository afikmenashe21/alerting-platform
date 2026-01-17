# Alerting Platform – System Patterns

## Control plane vs Data plane
- **Control plane**: rule-service (CRUD) + DB consistency, versioning.
- **Data plane**: evaluator/aggregator/sender handle high-throughput stream processing.

## Idempotency boundary (most important pattern)
- Dedupe is **not** done in evaluator (no shared state, high throughput).
- Dedupe is done in aggregator by persisting to `notifications` with unique key:
  - unique `(client_id, alert_id)`
- If Kafka redelivers the same `alerts.matched` message:
  - insert conflicts → NO duplicate notification emitted.

## Rule distribution & warm start
- rule-updater rebuilds a **Redis snapshot** of rule indexes + increments `rules:version`.
- evaluator on startup:
  1) reads snapshot from Redis and builds in-memory maps
  2) polls `rules:version` (or consumes `rule.changed`) to refresh snapshot when version changes

## Fast matching via inverted indexes + intersection
To match alert `(severity, source, name)` against rules:
- Maintain 3 maps:
  - `bySeverity[severity] -> []ruleInt`
  - `bySource[source] -> []ruleInt`
  - `byName[name] -> []ruleInt`
- Intersect candidate sets starting with smallest list to minimize work.
- Map `ruleInt -> clientInt` once (saves memory).

## Delivery responsibility separation
- Aggregator persists intents (RECEIVED) and emits work IDs.
- Sender does side-effects (email) and status updates.
- Exactly-once email is not possible; we aim for:
  - **idempotent record creation**
  - best-effort idempotent send (no-op if already SENT)

## Failure handling (MVP)
- evaluator crash before producing → Kafka re-deliver → safe.
- aggregator crash after insert before commit → Kafka re-deliver → insert idempotent.
- sender crash after sending but before status update → may re-send on retry; mitigate with provider idempotency key later.

## Partitioning conventions
- `alerts.new` key: `alert_id` (even distribution)
- `alerts.matched` key: `client_id` (tenant locality for DB shard)
- `notifications.ready` key: `client_id` or `notification_id` (either OK for MVP)
- `rule.changed` key: `rule_id`
