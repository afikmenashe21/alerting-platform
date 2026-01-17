# alert-producer – Progress

## Completed
- [x] Fixed-rate publishing with configurable RPS
- [x] Burst mode for stress testing
- [x] Alert generator with configurable distributions (severity/source/name)
- [x] Deterministic seed mode for reproducible tests
- [x] Kafka producer with alert_id keying
- [x] JSON message format with schema_version and event_ts
- [x] Graceful shutdown handling
- [x] CLI flags for all configuration options
- [x] Makefile for build/run/test
- [x] README with usage examples
- [x] Modular architecture with processor pattern

## Architecture Decisions

### Modular Architecture with Processor Pattern
- **Processor Pattern**: Main processing logic extracted into `internal/processor` package
- **Separation of Concerns**: 
  - `cmd/alert-producer/main.go`: CLI entry point, initialization, and orchestration
  - `internal/processor`: Business logic for different execution modes (burst, continuous, test)
  - `internal/generator`: Alert generation logic
  - `internal/producer`: Kafka publishing abstraction
- **Mode Handlers**: Separate methods for burst, continuous, and test modes
- **Extensibility**: Easy to add new execution modes by extending the processor

### Directory Structure
```
cmd/alert-producer/
└── main.go              # CLI entry point, initialization
internal/
├── processor/           # Processing orchestration
│   └── processor.go    # Main processor with mode handlers
├── generator/          # Alert generation
├── producer/          # Kafka publishing
└── config/            # Configuration
```

## Implementation Details
- Uses `github.com/segmentio/kafka-go` for Kafka producer
- Messages keyed by `alert_id` for even partition distribution
- Synchronous writes with `RequireOne` ack for at-least-once semantics
- Rate limiting via ticker with configurable interval
- Progress logging every 5 seconds in continuous mode
- Processor coordinates between generator and producer
