# Rule Changed Event Publishing - Validation Summary

## ✅ Validation Complete

The rule-service **correctly publishes** `rule.changed` events to Kafka for all rule mutations. Here's the validation:

### Event Publishing Points Verified

| Operation | HTTP Endpoint | Handler Method | Action Published | Status |
|-----------|--------------|----------------|------------------|--------|
| Create Rule | `POST /api/v1/rules` | `CreateRule()` | `CREATED` | ✅ Verified |
| Update Rule | `PUT /api/v1/rules/update` | `UpdateRule()` | `UPDATED` | ✅ Verified |
| Disable Rule | `POST /api/v1/rules/toggle` (enabled=false) | `ToggleRuleEnabled()` | `DISABLED` | ✅ Verified |
| Enable Rule | `POST /api/v1/rules/toggle` (enabled=true) | `ToggleRuleEnabled()` | `UPDATED` | ✅ Verified |
| Delete Rule | `DELETE /api/v1/rules/delete` | `DeleteRule()` | `DELETED` | ✅ Verified |

### Event Structure

All events follow this structure:

```json
{
  "rule_id": "550e8400-e29b-41d4-a716-446655440000",
  "client_id": "client-1",
  "action": "CREATED|UPDATED|DELETED|DISABLED",
  "version": 1,
  "updated_at": 1705257600,
  "schema_version": 1
}
```

### Kafka Message Configuration

- **Topic**: `rule.changed`
- **Partition Key**: `rule_id` (ensures same rule goes to same partition)
- **Headers**: `schema_version`, `action`, `rule_id`
- **Delivery**: At-least-once (synchronous write, waits for leader ack)
- **Timestamp**: Set from `updated_at` field

### Code Locations

1. **Event Definition**: `internal/events/events.go`
2. **Producer**: `internal/producer/producer.go` (Publish method)
3. **Event Publishing**:
   - Create: `internal/handlers/handlers.go:182-194`
   - Update: `internal/handlers/handlers.go:314-326`
   - Toggle: `internal/handlers/handlers.go:373-391`
   - Delete: `internal/handlers/handlers.go:425-437`

### Consumer Integration

#### rule-updater Service
- **Consumes**: `rule.changed` topic from Kafka
- **Behavior**: On any event, rebuilds full Redis snapshot:
  1. Queries all enabled rules from Postgres
  2. Builds compressed indexes (dictionaries + inverted indexes)
  3. Writes to Redis `rules:snapshot`
  4. Increments `rules:version` in Redis

#### evaluator Service
- **Consumes**: Indirectly via Redis version polling
- **Behavior**: 
  1. Polls `rules:version` periodically
  2. When version changes, reloads snapshot from Redis
  3. Atomically swaps in-memory indexes

### Event Flow

```
┌─────────────────┐
│  rule-service   │
│  (HTTP API)     │
└────────┬────────┘
         │
         │ DB Commit Success
         ▼
┌─────────────────┐
│  Publish Event  │
│  rule.changed   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Kafka Topic    │
│  rule.changed   │
└────────┬────────┘
         │
         ├──────────────────┐
         │                  │
         ▼                  ▼
┌──────────────┐    ┌──────────────┐
│ rule-updater │    │  evaluator  │
│              │    │  (via Redis) │
│ Consumes     │    │              │
│ Events       │    │ Polls        │
│              │    │ Version      │
│ → Rebuilds   │    │              │
│   Snapshot   │    │ → Reloads    │
│ → Updates     │    │   Snapshot   │
│   Redis      │    │ → Updates    │
│              │    │   Indexes    │
└──────────────┘    └──────────────┘
```

### Testing

Run the validation test:

```bash
make test-events
```

Or manually test:

```bash
# 1. Start the service
make run-all

# 2. In another terminal, create a rule
curl -X POST http://localhost:8081/api/v1/clients \
  -H "Content-Type: application/json" \
  -d '{"client_id": "test-client", "name": "Test"}'

curl -X POST http://localhost:8081/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "test-client",
    "severity": "HIGH",
    "source": "api",
    "name": "test"
  }'

# 3. Check rule-service logs for "Published rule changed event"
# 4. Consume from Kafka to verify:
docker exec <kafka-container> kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic rule.changed \
  --from-beginning \
  --max-messages 1
```

### Key Guarantees

✅ **Events published after DB commits** - Ensures data consistency  
✅ **All rule mutations trigger events** - CREATED, UPDATED, DELETED, DISABLED  
✅ **Events include all required fields** - rule_id, client_id, action, version, updated_at, schema_version  
✅ **Proper error handling** - Publishing failures are logged but don't fail operations  
✅ **Partition distribution** - Events keyed by rule_id for tenant locality  

### Notes

- Events are published **after** successful DB commits (not in transaction)
- If event publishing fails, the operation still succeeds (logged error)
- This is MVP approach; outbox pattern can be added later for stronger guarantees
- Events are keyed by `rule_id` ensuring same rule always goes to same partition
