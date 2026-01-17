# Evaluator Service

The evaluator service consumes alerts from Kafka, matches them against rules using in-memory indexes, and publishes matched alerts grouped by client.

## Overview

The evaluator is a stateless, high-throughput service that:
- Consumes `alerts.new` events from Kafka
- Matches alerts against rules using fast in-memory inverted indexes
- Publishes `alerts.matched` events to Kafka with matches grouped by client
- Hot-reloads rule indexes when rules change (via Redis version polling)

## Documentation

- **[Architecture](docs/ARCHITECTURE.md)** - Detailed service architecture, design patterns, and core concepts

## Building

```bash
make build
```

Or manually:
```bash
go build -o bin/evaluator ./cmd/evaluator
```

## Running

```bash
make run ARGS="-kafka-brokers localhost:9092 -redis-addr localhost:6379"
```

### Command-line Flags

- `-kafka-brokers`: Kafka broker addresses (comma-separated, default: `localhost:9092`)
- `-alerts-new-topic`: Kafka topic for incoming alerts (default: `alerts.new`)
- `-alerts-matched-topic`: Kafka topic for matched alerts (default: `alerts.matched`)
- `-consumer-group-id`: Kafka consumer group ID (default: `evaluator-group`)
- `-redis-addr`: Redis server address (default: `localhost:6379`)
- `-version-poll-interval`: Interval for polling Redis version (default: `5s`)

### Example

```bash
./bin/evaluator \
  -kafka-brokers localhost:9092 \
  -redis-addr localhost:6379 \
  -version-poll-interval 10s
```

## Dependencies

- **Kafka**: For consuming `alerts.new` and producing `alerts.matched`
- **Redis**: For loading rule snapshots and polling version changes

## Event Formats

### Input: `alerts.new`

```json
{
  "alert_id": "550e8400-e29b-41d4-a716-446655440000",
  "schema_version": 1,
  "event_ts": 1705257600,
  "severity": "HIGH",
  "source": "api",
  "name": "timeout",
  "context": {
    "environment": "prod",
    "region": "us-east-1"
  }
}
```

### Output: `alerts.matched`

**One message per client_id** (partitioned by client_id for tenant locality):

```json
{
  "alert_id": "550e8400-e29b-41d4-a716-446655440000",
  "schema_version": 1,
  "event_ts": 1705257600,
  "severity": "HIGH",
  "source": "api",
  "name": "timeout",
  "context": {
    "environment": "prod",
    "region": "us-east-1"
  },
  "client_id": "client-123",
  "rule_ids": ["rule-456", "rule-789"]
}
```

If an alert matches multiple clients, multiple messages are published (one per client).

## Key Properties

- **Stateless**: No deduplication (handled by aggregator)
- **Fast**: In-memory indexes with intersection algorithm
- **Hot-reloadable**: Polls Redis for rule updates without restart
- **At-least-once**: Kafka consumer commits offsets after processing

## Testing

```bash
make test
```

## Quick Start

The fastest way to get started:

```bash
make run-all
```

This single command will:
- ✅ Verify Docker is running
- ✅ Verify centralized infrastructure (Kafka, Redis) is available
- ✅ Verify connectivity to Kafka and Redis
- ✅ Download/update Go dependencies
- ✅ Create test rule snapshot if missing (for testing)
- ✅ Build the evaluator
- ✅ Run the evaluator

**Note:** Infrastructure (Kafka, Redis, Postgres) is managed centrally. Start it from the root directory:
```bash
cd ../.. && make setup-infra
```

### Manual Setup

If you prefer step-by-step setup:

1. **Ensure centralized infrastructure is running:**
   ```bash
   cd ../.. && make setup-infra
   ```

2. **Create test rule snapshot (optional, for testing):**
   ```bash
   make create-test-snapshot
   ```
   This creates 3 test rules for development. In production, `rule-updater` creates snapshots from the database.

3. **Build and run:**
   ```bash
   make build
   make run
   ```

### Testing the Pipeline

1. Produce alerts using alert-producer:
   ```bash
   cd ../alert-producer
   make run ARGS="-burst 10"
   ```

2. Watch evaluator logs - it should process alerts and publish matches

3. Check matched alerts in Kafka:
   ```bash
   docker exec alerting-platform-kafka kafka-console-consumer \
     --bootstrap-server localhost:9092 \
     --topic alerts.matched \
     --from-beginning \
     --max-messages 5
   ```

## Troubleshooting

### "Failed to connect to Redis"
- Ensure centralized infrastructure is running: `cd ../.. && make setup-infra`
- Verify Redis container: `docker ps | grep redis`
- Check Redis logs: `docker logs alerting-platform-redis`

### "Failed to load initial snapshot"
- The snapshot doesn't exist in Redis yet
- Run `rule-updater` service to create snapshots from the database
- Or create a test snapshot: `make create-test-snapshot`

### "Failed to create Kafka consumer"
- Ensure centralized infrastructure is running: `cd ../.. && make setup-infra`
- Verify Kafka container: `docker ps | grep kafka`
- Wait 10-15 seconds for Kafka to initialize
- Check Kafka logs: `docker logs alerting-platform-kafka`

### Infrastructure Issues
- All infrastructure is managed centrally from the root directory
- See `docs/architecture/INFRASTRUCTURE.md` for details
- Use `cd ../.. && make setup-infra` to start all infrastructure
