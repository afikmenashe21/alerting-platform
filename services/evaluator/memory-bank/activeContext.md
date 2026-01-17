# evaluator – Active Context

## Implemented
- ✅ Alert/rule in-memory structs defined in `internal/events` and `internal/snapshot`
- ✅ Snapshot loader from Redis (`internal/snapshot`)
- ✅ Intersection-based matcher (`internal/indexes`, `internal/matcher`)
- ✅ Grouping: client_id -> []rule_id (in matcher output)
- ✅ Kafka consume/produce loop (`internal/consumer`, `internal/producer`, `cmd/evaluator/main.go`)
- ✅ Version polling + hot reload (`internal/reloader`)
- ✅ One message per client_id output format (keyed by client_id for tenant locality)
- ✅ Automatic topic creation for `alerts.matched`
- ✅ Test snapshot script for development/testing

## Key Design Decisions
- **Output format**: One message per client_id (not one message with all matches)
  - Enables tenant locality in aggregator (messages partitioned by client_id)
  - Simplifies aggregator processing (one client per message)
- **Partitioning**: Messages keyed by `client_id` for tenant locality
- **Consumer offset**: Starts from beginning if no committed offset exists

## Next steps
- Integration with aggregator service
- Performance testing with high alert volumes

## Architecture Notes
- **No database dependency**: The evaluator only reads from Redis (`rules:snapshot` and `rules:version`)
- **Rule-updater responsibility**: The `rule-updater` service reads from the database and writes snapshots to Redis
- **Evaluator responsibility**: The evaluator only reads from Redis, never directly from the database
