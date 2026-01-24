# Aggregator

Deduplicates matched alerts into notifications using an idempotent Postgres insert, ensuring each `(client_id, alert_id)` pair produces exactly one notification.

## Role in Pipeline

```
alerts.matched (Kafka) → [aggregator] → notifications table (Postgres)
                                       → notifications.ready (Kafka)
```

The aggregator is the **idempotency boundary** of the platform. It converts the at-least-once Kafka delivery into exactly-once notification creation.

## How It Works

1. Consumes `alerts.matched` messages from Kafka
2. Attempts `INSERT ... ON CONFLICT DO NOTHING RETURNING notification_id` into the `notifications` table
3. If the insert succeeds (new notification): publishes a `notifications.ready` event
4. If the insert is a no-op (duplicate): skips publish, no side effects
5. Commits Kafka offset only after the DB operation succeeds

The unique constraint on `(client_id, alert_id)` is the dedup key. Kafka redeliveries after crashes are safe because the insert is idempotent.

## Performance

- ~50 notifications/s per instance (5.0 ms avg latency, DB-bound)
- Scales horizontally: each instance joins the same consumer group
- Bottleneck is Postgres write throughput

## Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `-kafka-brokers` | `localhost:9092` | Kafka broker addresses |
| `-alerts-matched-topic` | `alerts.matched` | Input topic |
| `-notifications-ready-topic` | `notifications.ready` | Output topic |
| `-consumer-group-id` | `aggregator-group` | Kafka consumer group |
| `-postgres-dsn` | `postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable` | Postgres connection string |

## Events

### Input: `alerts.matched`

```json
{
  "alert_id": "alert-123",
  "severity": "HIGH",
  "source": "monitoring",
  "name": "cpu_high",
  "context": {"host": "server1"},
  "client_id": "client-456",
  "rule_ids": ["rule-789", "rule-790"]
}
```

### Output: `notifications.ready`

```json
{
  "notification_id": "550e8400-...",
  "client_id": "client-456",
  "alert_id": "alert-123"
}
```

## Database Schema

The `notifications` table:

| Column | Type | Notes |
|--------|------|-------|
| `notification_id` | UUID | Primary key (DB-generated) |
| `client_id` | VARCHAR | Part of unique constraint |
| `alert_id` | VARCHAR | Part of unique constraint |
| `severity` | VARCHAR | Alert severity |
| `source` | VARCHAR | Alert source |
| `name` | VARCHAR | Alert name |
| `context` | JSONB | Optional alert context |
| `rule_ids` | TEXT[] | All matching rule IDs |
| `status` | VARCHAR | `RECEIVED` or `SENT` |
| `created_at` | TIMESTAMP | - |

**Unique constraint**: `(client_id, alert_id)` — the idempotency key.

Migration: `000006_create_notifications_table.up.sql`

## Running

```bash
# From project root: start infrastructure first
make setup-infra && make run-migrations

# Then run this service
cd services/aggregator
make run-all
```

## Testing

```bash
make test
```

## Key Properties

- **Idempotent**: Duplicate Kafka messages produce no duplicate notifications
- **At-least-once**: Commits offset only after successful DB write
- **Crash-safe**: If crash occurs after insert but before offset commit, redelivery is a no-op
- **Horizontally scalable**: Multiple instances share partitions via consumer group
