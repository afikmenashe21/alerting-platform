# Rule-Updater Service

The **rule-updater** service is responsible for maintaining a Redis snapshot of all enabled rules. It consumes `rule.changed` events from Kafka and rebuilds the snapshot whenever rules are created, updated, deleted, or disabled.

## Overview

The rule-updater service:
1. Consumes `rule.changed` events from Kafka
2. Queries all enabled rules from Postgres
3. Builds an optimized snapshot with inverted indexes for fast matching
4. Writes the snapshot to Redis and increments the version
5. Enables the evaluator service to warm-start quickly without querying the database

## Architecture

```
rule-service (CRUD) → Kafka (rule.changed) → rule-updater → Redis (snapshot + version)
                                                                    ↓
                                                          evaluator (warm-start)
```

## Documentation

- **[Architecture](docs/ARCHITECTURE.md)** - Service architecture and design patterns
- **[Test Coverage](docs/TEST_COVERAGE.md)** - Test coverage documentation and running tests

## Quick Start

### Prerequisites

- Go 1.22+
- Docker and Docker Compose
- Postgres database with rules table (from rule-service migrations)
- Kafka with `rule.changed` topic
- Redis

### Run Everything

The easiest way to get started:

```bash
make run-all
```

This script will:
1. Check dependencies (Go, Docker)
2. Download Go dependencies
3. Start required services (Postgres, Redis, Kafka)
4. Create Kafka topics
5. Build and run the service

### Manual Setup

1. **Start dependencies:**
   ```bash
   make up
   ```

2. **Create Kafka topics:**
   ```bash
   make create-topics
   ```

3. **Build and run:**
   ```bash
   make build
   make run
   ```

## Configuration

The service accepts the following command-line flags:

- `-kafka-brokers`: Kafka broker addresses (default: `localhost:9092`)
- `-rule-changed-topic`: Kafka topic for rule change events (default: `rule.changed`)
- `-consumer-group-id`: Kafka consumer group ID (default: `rule-updater-group`)
- `-postgres-dsn`: PostgreSQL connection string (default: `postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable`)
- `-redis-addr`: Redis server address (default: `localhost:6379`)

### Example

```bash
./bin/rule-updater \
  -kafka-brokers localhost:9092 \
  -rule-changed-topic rule.changed \
  -consumer-group-id rule-updater-group \
  -postgres-dsn postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable \
  -redis-addr localhost:6379
```

## How It Works

### Initial Snapshot

On startup, the service:
1. Queries all enabled rules from Postgres
2. Builds a snapshot with:
   - **Dictionaries**: Maps strings (severity, source, name) to integers for compression
   - **Inverted indexes**: Maps field values to rule integers for fast matching
   - **Rule mapping**: Maps rule integers to rule IDs and client IDs
3. Writes the snapshot to Redis at key `rules:snapshot`
4. Sets/increments the version at key `rules:version`

### Event Processing

When a `rule.changed` event is received:
1. The service queries **all enabled rules** from Postgres (not just the changed one)
2. Rebuilds the complete snapshot
3. Writes the new snapshot to Redis
4. Increments the version
5. Commits the Kafka offset

This approach ensures consistency: the snapshot always reflects the complete state of all enabled rules.

### Snapshot Format

The snapshot stored in Redis is a JSON object with the following structure:

```json
{
  "schema_version": 1,
  "severity_dict": {"HIGH": 1, "MEDIUM": 2, "LOW": 3},
  "source_dict": {"api": 1, "db": 2},
  "name_dict": {"timeout": 1, "error": 2},
  "by_severity": {"HIGH": [1, 3], "MEDIUM": [2]},
  "by_source": {"api": [1, 3], "db": [2]},
  "by_name": {"timeout": [1, 3], "error": [2]},
  "rules": {
    "1": {"rule_id": "rule-001", "client_id": "client-1"},
    "2": {"rule_id": "rule-002", "client_id": "client-1"},
    "3": {"rule_id": "rule-003", "client_id": "client-2"}
  }
}
```

## Makefile Targets

- `make build` - Build the binary
- `make run` - Run the service
- `make test` - Run tests
- `make clean` - Remove build artifacts
- `make deps` - Download dependencies
- `make up` - Start Docker services (Kafka, Postgres, Redis)
- `make down` - Stop Docker services
- `make logs` - Show service logs
- `make status` - Show service status
- `make create-topics` - Create Kafka topics
- `make setup` - Start services and create topics
- `make run-all` - Complete setup and run (recommended)

## Testing

### Running Tests

See **[Test Coverage Documentation](docs/TEST_COVERAGE.md)** for detailed information about test coverage and running tests.

Quick start:
```bash
# Install test dependencies
go get github.com/DATA-DOG/go-sqlmock
go mod tidy

# Run all tests
go test ./... -v

# Run with coverage
go test ./... -cover

# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Manual Testing

1. **Start the service:**
   ```bash
   make run-all
   ```

2. **Create a rule via rule-service:**
   ```bash
   curl -X POST http://localhost:8081/api/v1/rules \
     -H "Content-Type: application/json" \
     -d '{
       "client_id": "client-1",
       "severity": "HIGH",
       "source": "api",
       "name": "timeout"
     }'
   ```

3. **Check Redis snapshot:**
   ```bash
   docker exec alerting-platform-redis redis-cli GET rules:snapshot | jq .
   docker exec alerting-platform-redis redis-cli GET rules:version
   ```

4. **Verify the service logs** show the snapshot being rebuilt

## Integration with Other Services

- **rule-service**: Publishes `rule.changed` events that trigger snapshot rebuilds
- **evaluator**: Reads the snapshot from Redis for warm-start and watches for version changes

## Error Handling

- **Database connection errors**: Service exits with error message and tips
- **Redis connection errors**: Service exits with error message and tips
- **Kafka read errors**: Logs error and continues processing (at-least-once semantics)
- **Snapshot build errors**: Logs error, doesn't commit offset (Kafka will redeliver)

## At-Least-Once Semantics

The service uses at-least-once delivery semantics:
- Offsets are committed only after successfully rebuilding and writing the snapshot
- If the service crashes before committing, Kafka will redeliver the event
- Rebuilding the snapshot is idempotent (always queries current state from DB)

## Development

### Project Structure

```
rule-updater/
├── cmd/
│   └── rule-updater/
│       └── main.go          # Entry point
├── internal/
│   ├── config/              # Configuration parsing
│   ├── consumer/            # Kafka consumer
│   ├── database/            # Postgres operations
│   ├── events/              # Event structures
│   └── snapshot/             # Snapshot building and writing
├── scripts/
│   └── run-all.sh           # Setup and run script
├── docker-compose.yml       # Centralized infrastructure (at root level)
├── Makefile                 # Build and run targets
├── go.mod                   # Go dependencies
└── README.md                # This file
```

## Troubleshooting

### Service won't start

- **Check Postgres is running**: `docker ps | grep postgres`
- **Check Redis is running**: `docker ps | grep redis`
- **Check Kafka is running**: `docker ps | grep kafka`
- **Check ports are available**: `lsof -i :5432`, `lsof -i :6379`, `lsof -i :9092`

### No snapshot in Redis

- **Check service logs**: Look for errors during snapshot build
- **Verify rules exist**: Query Postgres to ensure there are enabled rules
- **Check Redis connection**: Verify Redis is accessible

### Snapshot not updating

- **Check Kafka topic**: Verify `rule.changed` topic exists and has messages
- **Check consumer group**: Verify consumer is reading from the topic
- **Check service logs**: Look for errors during event processing
