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

## Scaling patterns

### Horizontal scaling (free)
- **Kafka partitions**: Set `KAFKA_NUM_PARTITIONS` for auto-created topics
  - More partitions = more parallel consumers
  - 6 partitions allows up to 6 consumers per consumer group
- **ECS task count**: Increase `desired_count` in Terraform
  - Evaluator and aggregator scale horizontally
  - Each instance joins same Kafka consumer group
  - Kafka rebalances partitions across instances

### Rate limiting patterns
- **Email provider rate limiting**: Token bucket at provider registry level
  - Prevents external API rate limit errors (e.g., Resend 2 RPS)
  - Applied centrally, not per-worker
  - Configurable via `EMAIL_RATE_LIMIT` env var
- **Test email filtering**: Skip sending to test domains
  - `@example.com`, `@test.com`, `@localhost`, etc.
  - Prevents wasted quota on test data

### Memory optimization
- **Container sizing**: Balance memory per task vs task count
  - t3.small: ~1900 MB usable for containers
  - 150 MB per service allows 10+ containers
- **Heap tuning**: Kafka/Zookeeper heap sizes reduced for low-memory deployment
  - Zookeeper: 128 MB heap
  - Kafka: 256 MB heap

### Bottleneck identification
1. **Single Kafka broker**: Limits write throughput (~800/s)
2. **Memory constraint**: Limits number of task instances
3. **Database connections**: PostgreSQL connection pool limits
