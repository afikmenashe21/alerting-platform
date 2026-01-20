# aggregator – Progress

- [x] notifications schema + unique index
- [x] idempotent insert + return notification_id
- [x] produce notifications.ready
- [x] correct offset commit ordering
- [x] Modular architecture with processor pattern
- [x] Code cleanup and modularization:
  - Extracted shared Kafka validation helpers (`ValidateConsumerParams`, `ValidateProducerParams`)
  - Extracted Kafka constants (`ReadTimeout`, `CommitInterval`, `WriteTimeout`)
  - Extracted `NewReaderConfig` helper for standardized Kafka reader configuration
  - Extracted `buildMessage` helper from producer for message construction
  - Extracted `marshalContextToJSONB` helper from database for JSON marshaling
  - Added `NewNotificationReady` helper function for event building
  - Removed duplicate validation logic from consumer and producer
  - All tests pass; behavior unchanged

## Architecture Decisions

### Modular Architecture with Processor Pattern
- **Processor Pattern**: Main processing logic extracted into `internal/processor` package
- **Separation of Concerns**:
  - `cmd/aggregator/main.go`: CLI entry point, initialization, and orchestration
  - `internal/processor`: Business logic for notification aggregation and deduplication
  - `internal/database`: Data access layer with idempotent insert operations
  - `internal/consumer`: Kafka consumer abstraction
  - `internal/producer`: Kafka producer abstraction
- **Deduplication Boundary**: Processor enforces idempotent inserts at the database level
- **Extensibility**: Easy to add new processing logic or deduplication strategies

### Directory Structure
```
cmd/aggregator/
└── main.go              # CLI entry point, initialization
internal/
├── processor/           # Processing orchestration
│   └── processor.go    # Notification aggregation logic
├── database/           # Data access layer
├── consumer/          # Kafka consumer
├── producer/           # Kafka producer
├── events/            # Event structures and builders
├── kafka/             # Shared Kafka utilities and validation
└── config/            # Configuration
```

## Implementation Details

- Database migrations created in `migrations/` directory
- Idempotent insert uses `INSERT ... ON CONFLICT DO NOTHING RETURNING`
- Unique constraint on `(client_id, alert_id)` enforces dedupe boundary
- Offset commit only after successful DB operation and (if applicable) publish
- Uses `pq.Array` for proper PostgreSQL array handling
- Processor coordinates between consumer, database, and producer
- Shared Kafka utilities in `internal/kafka` package:
  - `ValidateConsumerParams`, `ValidateProducerParams`: Common validation
  - `ParseBrokers`: Broker list parsing
  - `NewReaderConfig`: Standardized reader configuration
  - Constants: `ReadTimeout`, `CommitInterval`, `WriteTimeout`
- Event builders: `NewNotificationReady` helper for constructing events
- Message building: `buildMessage` helper extracts message construction logic
- JSON marshaling: `marshalContextToJSONB` helper for database context serialization
