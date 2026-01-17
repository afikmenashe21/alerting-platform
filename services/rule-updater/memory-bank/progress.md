# rule-updater – Progress

## Completed
- [x] consume rule.changed from Kafka
- [x] initial snapshot build on startup (queries all enabled rules from DB)
- [x] write rules:snapshot to Redis (with dictionaries and inverted indexes)
- [x] bump rules:version atomically with snapshot write
- [x] incremental snapshot updates based on rule.changed events
- [x] graceful shutdown handling
- [x] at-least-once semantics (commit offset after successful snapshot write)
- [x] Modular architecture with processor pattern

## Architecture Decisions

### Modular Architecture with Processor Pattern
- **Processor Pattern**: Main processing logic extracted into `internal/processor` package
- **Separation of Concerns**:
  - `cmd/rule-updater/main.go`: CLI entry point, initialization, and orchestration
  - `internal/processor`: Business logic for processing rule changes and updating snapshots
  - `internal/snapshot`: Snapshot building and Redis operations
  - `internal/database`: Data access layer
  - `internal/consumer`: Kafka consumer abstraction
- **Incremental Updates**: Processor handles rule change events and applies updates directly to Redis
- **Extensibility**: Easy to add new processing logic or snapshot update strategies

### Directory Structure
```
cmd/rule-updater/
└── main.go              # CLI entry point, initialization
internal/
├── processor/           # Processing orchestration
│   └── processor.go     # Rule change processing logic
├── snapshot/            # Snapshot building and Redis operations
├── database/           # Data access layer
├── consumer/           # Kafka consumer
└── config/            # Configuration
```

## Implementation Details
- **Snapshot format**: Matches evaluator's expected format with dictionaries (severity_dict, source_dict, name_dict) and inverted indexes (bySeverity, bySource, byName)
- **Rule integers**: Each rule gets a unique integer (ruleInt) starting from 1, used in inverted indexes
- **Atomic updates**: Snapshot and version are updated together using Redis pipeline
- **Incremental updates**: 
  - Processor applies incremental update based on action (CREATED/UPDATED/DELETED/DISABLED)
  - For CREATED/UPDATED: Fetches rule from DB and adds/updates in Redis directly
  - For DELETED/DISABLED: Removes rule from Redis directly
- **Database methods**: GetRule() to fetch single rule by ID for incremental updates
- **Snapshot methods**: LoadSnapshot(), AddRule(), UpdateRule(), RemoveRule()
- **Dependencies**: Postgres (for rules), Redis (for snapshot storage), Kafka (for rule.changed events)
