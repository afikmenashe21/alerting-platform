# Rule Updater

Maintains a Redis snapshot of all enabled rules with inverted indexes for fast matching. Rebuilds the snapshot whenever rules change.

## Role in Pipeline

```
rule.changed (Kafka) → [rule-updater] → Redis (rules:snapshot + rules:version)
                             ↑                         ↓
                      Postgres (rules)          evaluator (warm start)
```

The rule-updater bridges the control plane (rule-service + Postgres) and the data plane (evaluator + Redis). It ensures the evaluator can start quickly without querying the database.

## How It Works

1. On startup, queries all enabled rules from Postgres and builds an initial snapshot
2. Writes the snapshot to Redis key `rules:snapshot` and increments `rules:version`
3. Consumes `rule.changed` events from Kafka
4. On each event, rebuilds the **complete snapshot** from Postgres (not incremental)
5. Writes the new snapshot and increments the version
6. Commits Kafka offset only after successful Redis write

The full-rebuild approach ensures consistency: the snapshot always reflects the complete current state, regardless of missed or reordered events.

## Snapshot Format

The snapshot is a JSON object stored in Redis with inverted indexes:

```json
{
  "schema_version": 1,
  "severity_dict": {"HIGH": 1, "MEDIUM": 2},
  "source_dict": {"api": 1, "db": 2},
  "name_dict": {"timeout": 1, "error": 2},
  "by_severity": {"HIGH": [1, 3], "MEDIUM": [2]},
  "by_source": {"api": [1], "db": [2, 3]},
  "by_name": {"timeout": [1], "error": [2, 3]},
  "rules": {
    "1": {"rule_id": "rule-001", "client_id": "client-1"},
    "2": {"rule_id": "rule-002", "client_id": "client-1"}
  }
}
```

Dictionaries map string values to integers for compression. Inverted indexes map field values to lists of rule integers for O(1) lookup.

## Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `-kafka-brokers` | `localhost:9092` | Kafka broker addresses |
| `-rule-changed-topic` | `rule.changed` | Input topic |
| `-consumer-group-id` | `rule-updater-group` | Kafka consumer group |
| `-postgres-dsn` | `postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable` | Postgres connection string |
| `-redis-addr` | `localhost:6379` | Redis address |

## Events

### Input: `rule.changed`

```json
{
  "rule_id": "uuid",
  "client_id": "client-1",
  "action": "CREATED|UPDATED|DELETED|DISABLED",
  "version": 1,
  "updated_at": 1234567890
}
```

## Running

```bash
# From project root: start infrastructure first
make setup-infra && make run-migrations

# Then run this service
cd services/rule-updater
make run-all
```

### Verify Snapshot

```bash
# Check snapshot exists
docker exec alerting-platform-redis redis-cli GET rules:version

# View snapshot content
docker exec alerting-platform-redis redis-cli GET rules:snapshot | jq .
```

## Testing

```bash
make test
```

## Key Properties

- **Single instance only**: Must run as exactly 1 instance (writes a single Redis snapshot)
- **Full rebuild**: Always queries complete rule set from Postgres (not incremental)
- **Idempotent**: Rebuilding the same state produces the same snapshot
- **At-least-once**: Commits offset only after successful Redis write
