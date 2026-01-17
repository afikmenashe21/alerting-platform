# Rule Changed Event Validation

This document validates that `rule.changed` events are properly published to Kafka and can be consumed by evaluator and rule-updater services.

## Event Publishing Points

The rule-service publishes `rule.changed` events in the following scenarios:

### 1. Rule Created (`CREATED`)
- **Trigger**: `POST /api/v1/rules`
- **Handler**: `CreateRule()`
- **Location**: `internal/handlers/handlers.go:182-194`
- **Event Fields**:
  - `rule_id`: UUID of the created rule
  - `client_id`: Client ID the rule belongs to
  - `action`: `"CREATED"`
  - `version`: Initial version (1)
  - `updated_at`: Unix timestamp
  - `schema_version`: 1

### 2. Rule Updated (`UPDATED`)
- **Trigger**: `PUT /api/v1/rules/update?rule_id=<id>`
- **Handler**: `UpdateRule()`
- **Location**: `internal/handlers/handlers.go:314-326`
- **Event Fields**:
  - `rule_id`: UUID of the updated rule
  - `client_id`: Client ID the rule belongs to
  - `action`: `"UPDATED"`
  - `version`: New version (incremented)
  - `updated_at`: Unix timestamp
  - `schema_version`: 1

### 3. Rule Disabled (`DISABLED`)
- **Trigger**: `POST /api/v1/rules/toggle?rule_id=<id>` with `enabled: false`
- **Handler**: `ToggleRuleEnabled()`
- **Location**: `internal/handlers/handlers.go:373-391`
- **Event Fields**:
  - `rule_id`: UUID of the rule
  - `client_id`: Client ID the rule belongs to
  - `action`: `"DISABLED"`
  - `version`: New version (incremented)
  - `updated_at`: Unix timestamp
  - `schema_version`: 1

### 4. Rule Re-enabled (`UPDATED`)
- **Trigger**: `POST /api/v1/rules/toggle?rule_id=<id>` with `enabled: true`
- **Handler**: `ToggleRuleEnabled()`
- **Location**: `internal/handlers/handlers.go:373-391`
- **Event Fields**:
  - `rule_id`: UUID of the rule
  - `client_id`: Client ID the rule belongs to
  - `action`: `"UPDATED"` (re-enabling is treated as update)
  - `version`: New version (incremented)
  - `updated_at`: Unix timestamp
  - `schema_version`: 1

### 5. Rule Deleted (`DELETED`)
- **Trigger**: `DELETE /api/v1/rules/delete?rule_id=<id>`
- **Handler**: `DeleteRule()`
- **Location**: `internal/handlers/handlers.go:425-437`
- **Event Fields**:
  - `rule_id`: UUID of the deleted rule
  - `client_id`: Client ID the rule belonged to
  - `action`: `"DELETED"`
  - `version`: Last known version
  - `updated_at`: Current Unix timestamp
  - `schema_version`: 1

## Event Structure

```json
{
  "rule_id": "550e8400-e29b-41d4-a716-446655440000",
  "client_id": "client-1",
  "action": "CREATED",
  "version": 1,
  "updated_at": 1705257600,
  "schema_version": 1
}
```

## Kafka Message Details

- **Topic**: `rule.changed`
- **Partition Key**: `rule_id` (for partition distribution)
- **Headers**:
  - `schema_version`: String representation of schema version
  - `action`: Action type (CREATED/UPDATED/DELETED/DISABLED)
  - `rule_id`: Rule ID for filtering/routing
- **Timestamp**: Set from `updated_at` field
- **Delivery**: At-least-once (synchronous write, waits for leader ack)

## Consumer Expectations

### rule-updater Service
- **Consumes**: `rule.changed` topic
- **Behavior**: On any `rule.changed` event, rebuilds full snapshot from DB:
  1. Loads all enabled rules from Postgres
  2. Builds dictionaries (strings → ints) for compression
  3. Builds inverted indexes (bySeverity/bySource/byName)
  4. Builds ruleInt → (rule_id, client_id) mapping
  5. Writes snapshot blob to Redis `rules:snapshot`
  6. Increments `rules:version` in Redis

### evaluator Service
- **Consumes**: Indirectly via Redis version polling
- **Behavior**: 
  1. Polls Redis `rules:version` periodically
  2. When version changes, reloads snapshot from Redis
  3. Builds in-memory indexes from snapshot
  4. Atomically swaps indexes for matching

## Validation Checklist

- [x] Events published after successful DB commits
- [x] Events include all required fields (rule_id, client_id, action, version, updated_at, schema_version)
- [x] Events keyed by rule_id for partition distribution
- [x] Events published for CREATED, UPDATED, DELETED, DISABLED actions
- [x] Error handling: Event publishing failures are logged but don't fail the operation
- [x] Event structure matches consumer expectations

## Testing

Run the test script to validate event publishing:

```bash
./scripts/test-rule-events.sh
```

Or manually test:

```bash
# 1. Create a client
curl -X POST http://localhost:8081/api/v1/clients \
  -H "Content-Type: application/json" \
  -d '{"client_id": "test-client", "name": "Test"}'

# 2. Create a rule (should publish CREATED event)
curl -X POST http://localhost:8081/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "test-client",
    "severity": "HIGH",
    "source": "api",
    "name": "test"
  }'

# 3. Check rule-service logs for "Published rule changed event"
# 4. Consume from Kafka to verify event:
docker exec <kafka-container> kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic rule.changed \
  --from-beginning \
  --max-messages 1
```

## Event Flow

```
rule-service (HTTP API)
  ↓ (DB commit successful)
  ↓ (Publish rule.changed event)
Kafka (rule.changed topic)
  ├──→ rule-updater (consumes event)
  │     └──→ Rebuilds Redis snapshot
  │           └──→ Increments rules:version
  │
  └──→ evaluator (polls Redis rules:version)
        └──→ Reloads snapshot when version changes
              └──→ Updates internal indexes
```

## Notes

- Events are published **after** successful DB commits (not in a transaction)
- If event publishing fails, the operation still succeeds (logged error)
- This is MVP approach; outbox pattern can be added later for stronger guarantees
- Events are keyed by `rule_id` for partition distribution (ensures same rule goes to same partition)
