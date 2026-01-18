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

## UI Integration (Planned)

### Integration with rule-service-ui
The alert-producer will be integrated into the rule-service-ui project to allow users to generate alerts directly from the web interface with optional manual configuration.

### Architecture
- **Dual Interface**: Maintain both CLI (`cmd/alert-producer`) and HTTP API (`cmd/alert-producer-api`)
- **Shared Core**: Both interfaces use the same internal packages (processor, generator, producer)
- **API Server**: New HTTP server wraps alert-producer functionality as REST endpoints
- **Job Management**: Track alert generation jobs with unique IDs for status monitoring

### Configuration Options
All CLI flags will be available via API:
- `rps`: Alerts per second (float)
- `duration`: Duration to run (string, e.g., "60s", "5m")
- `burst`: Burst mode - send N alerts immediately (int, 0 = continuous)
- `seed`: Random seed for deterministic generation (int64, 0 = random)
- `severity-dist`: Severity distribution (string, format: "HIGH:30,MEDIUM:30,...")
- `source-dist`: Source distribution (string, format: "api:25,db:20,...")
- `name-dist`: Name distribution (string, format: "timeout:15,error:15,...")
- `kafka-brokers`: Kafka broker addresses (string, default: "localhost:9092")
- `topic`: Kafka topic name (string, default: "alerts.new")
- `mock`: Use mock producer (boolean, default: false)
- `test`: Test mode flag (boolean, default: false)
- `single-test`: Single test alert mode (boolean, default: false)

### UI Component Design
- **Alert Generator Tab**: New tab in rule-service-ui alongside Clients, Rules, Endpoints
- **Preset Buttons**: Quick actions for common scenarios
- **Configuration Form**: Full form with all options for manual configuration
- **Status Display**: Real-time updates showing alerts sent, errors, progress
- **Job History**: List of previous generation runs with details

### Benefits
- **User-Friendly**: No need to use CLI or terminal
- **Visual Feedback**: See generation progress in real-time
- **Testing Workflow**: Easy to test rules by generating matching alerts
- **Load Testing**: Simple interface for performance testing
- **Accessibility**: Non-technical users can generate test alerts
