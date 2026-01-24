# Evaluator

Matches incoming alerts against customer rules using in-memory inverted indexes and publishes one match event per client.

## Role in Pipeline

```
alerts.new (Kafka) → [evaluator] → alerts.matched (Kafka)
                         ↑
                   Redis (rule snapshot)
```

The evaluator is a **stateless, high-throughput** data-plane service. It hot-reloads rule indexes from Redis without restart.

## How It Works

1. On startup, loads the rule snapshot from Redis into memory (warm start)
2. Polls `rules:version` in Redis to detect rule changes; rebuilds indexes when version increments
3. For each alert on `alerts.new`:
   - Looks up candidates in three inverted indexes: `bySeverity`, `bySource`, `byName`
   - Intersects candidate sets starting from the smallest (fast elimination)
   - Groups matching rules by `client_id`
   - Publishes one `alerts.matched` message per client (keyed by `client_id`)
4. Commits Kafka offset after successful publish

## Performance

- ~160 alerts/s per instance (3.2 ms avg latency)
- Scales horizontally: each instance joins the same consumer group
- Stateless: no DB writes, no shared mutable state

## Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `-kafka-brokers` | `localhost:9092` | Kafka broker addresses |
| `-alerts-new-topic` | `alerts.new` | Input topic |
| `-alerts-matched-topic` | `alerts.matched` | Output topic |
| `-consumer-group-id` | `evaluator-group` | Kafka consumer group |
| `-redis-addr` | `localhost:6379` | Redis address (for rule snapshot) |
| `-version-poll-interval` | `5s` | How often to check for rule updates |

## Events

### Input: `alerts.new`

```json
{
  "alert_id": "550e8400-...",
  "severity": "HIGH",
  "source": "api",
  "name": "timeout",
  "context": {"region": "us-east-1"}
}
```

### Output: `alerts.matched`

One message per matching client (keyed by `client_id`):

```json
{
  "alert_id": "550e8400-...",
  "severity": "HIGH",
  "source": "api",
  "name": "timeout",
  "context": {"region": "us-east-1"},
  "client_id": "client-123",
  "rule_ids": ["rule-456", "rule-789"]
}
```

## Running

```bash
# From project root: start infrastructure first
make setup-infra

# Then run this service
cd services/evaluator
make run-all
```

## Testing

```bash
make test
```

## Key Properties

- **Stateless**: No deduplication responsibility (handled by aggregator)
- **Hot-reloadable**: Picks up rule changes via Redis version polling
- **At-least-once**: Commits offset only after successful publish
- **Horizontally scalable**: Multiple instances share partitions via consumer group
