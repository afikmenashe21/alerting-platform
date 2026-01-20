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
- [x] Code cleanup and modularization:
  - Split 618-line `snapshot.go` into three focused files:
    - `snapshot.go` (267 lines): Core Snapshot struct and in-memory operations
    - `writer.go` (156 lines): Writer struct and Redis operations
    - `lua_scripts.go` (207 lines): Lua script constants for direct Redis updates
  - Extracted redundant code into helper functions:
    - `getMaxDictValue()`: Reusable dictionary max value calculation
    - `removeFromIndex()`: Unified index removal logic
    - `newEmptySnapshot()`: Centralized empty snapshot creation
  - All tests pass; behavior unchanged
- [x] Additional code cleanup and modularization:
  - Split `snapshot.go` (267 lines) into focused files:
    - `snapshot.go`: Types, constants, and helper functions (getMaxDictValue, removeFromIndex, removeFromSlice, newEmptySnapshot)
    - `builder.go`: BuildSnapshot function for initial snapshot construction
    - `operations.go`: Snapshot operations (AddRule, UpdateRule, RemoveRule) and helper methods (findRuleInt, getNextRuleInt, addToIndex, addToDictionaries, addToIndexes)
  - Extracted additional helper functions to reduce duplication:
    - `addToIndex()`: Helper for adding ruleInt to index maps
    - `addToDictionaries()`: Helper for adding values to dictionaries
    - `addToIndexes()`: Helper for adding ruleInt to all inverted indexes
  - All tests pass; behavior unchanged

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
│   ├── snapshot.go      # Core Snapshot struct and in-memory operations
│   ├── writer.go        # Writer struct and Redis operations
│   └── lua_scripts.go   # Lua script constants for direct Redis updates
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
