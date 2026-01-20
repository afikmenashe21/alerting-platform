# evaluator – Progress

- [x] Redis snapshot loader
- [x] In-memory indexes + intersection matcher
- [x] Group-by client output format (one message per client_id)
- [x] Kafka consume/produce loop
- [x] Version polling + hot reload
- [x] Automatic topic creation
- [x] Test snapshot script for development
- [x] Modular architecture with processor pattern

## Architecture Decisions

### Modular Architecture with Processor Pattern
- **Processor Pattern**: Main processing logic extracted into `internal/processor` package
- **Separation of Concerns**:
  - `cmd/evaluator/main.go`: CLI entry point, initialization, and orchestration
  - `internal/processor`: Business logic for alert evaluation and rule change handling
    - `processor.go`: Alert processing and matching logic
    - `rulehandler.go`: Rule change event handling
  - `internal/matcher`: Rule matching logic
  - `internal/indexes`: In-memory index management
  - `internal/snapshot`: Snapshot loading from Redis
  - `internal/reloader`: Hot reload mechanism
- **Dual Processing**: Separate handlers for alert processing and rule change events
- **Extensibility**: Easy to add new processing logic or matching strategies

### Directory Structure
```
cmd/evaluator/
└── main.go              # CLI entry point, initialization
internal/
├── processor/           # Processing orchestration
│   ├── processor.go    # Alert evaluation processing
│   └── rulehandler.go  # Rule change event handling
├── matcher/            # Rule matching logic
├── indexes/           # In-memory index management
├── snapshot/           # Snapshot loading
├── reloader/           # Hot reload mechanism
├── consumer/           # Kafka consumer
├── ruleconsumer/      # Rule change consumer
├── producer/          # Kafka producer
└── config/            # Configuration
```

## Implementation Details
- **Output**: One message per client_id (keyed by client_id for tenant locality)
- **Event structure**: `{alert, client_id, rule_ids[]}` per message
- **Partitioning**: Messages partitioned by client_id (hash of client_id)
- **Consumer**: Starts from beginning if no committed offset
- **Processor**: Coordinates alert consumption, matching, and publishing
- **Rule Handler**: Handles rule.changed events and triggers immediate reloads

## Code Cleanup and Modularization
- [x] **Extracted shared Kafka utilities**:
  - `internal/kafka/util.go`: Added `ValidateConsumerParams()` and `ValidateProducerParams()` for common validation
  - `internal/kafka/util.go`: Added `NewReaderConfig()` for standardized Kafka reader configuration
  - Removed duplicate validation logic from `consumer` and `ruleconsumer` packages
  - Removed duplicate validation logic from `producer` package
- [x] **Extracted event building helper**:
  - `internal/events/events.go`: Added `NewAlertMatched()` helper function to build AlertMatched from AlertNew
  - Simplified processor code by removing inline struct construction
- [x] **Code organization**:
  - All non-test files are under 200 lines (no splitting needed)
  - Test files remain large but are acceptable for comprehensive test coverage
  - All tests pass; behavior unchanged
