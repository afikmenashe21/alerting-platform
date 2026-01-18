# alert-producer

Generates synthetic alerts and publishes them to Kafka `alerts.new` topic for testing the alerting platform pipeline.

## 📚 Documentation

- **[Getting Started](docs/SETUP_AND_RUN.md)** - Complete setup and run guide
- **[HTTP API Server](docs/API_SERVER.md)** - REST API for UI integration
- **[Architecture](docs/STRUCTURE.md)** - Code organization and design decisions
- **[Event Structure](docs/EVENT_STRUCTURE.md)** - Alert event JSON schema
- **[Partitioning Strategy](docs/PARTITIONING.md)** - How we avoid hot partitions

See [docs/README.md](docs/README.md) for complete documentation index.

## Features

- **CLI Interface**: Command-line tool for generating alerts
- **HTTP API Server**: REST API for web UI integration (see [API_SERVER.md](docs/API_SERVER.md))
- **Configurable rate**: Set alerts per second (RPS)
- **Burst mode**: Send N alerts immediately for stress testing
- **Distributions**: Configure severity/source/name distributions
- **Rule matching**: Default distributions match test-data generator values to ensure alerts match existing rules
- **Deterministic mode**: Use seed for reproducible test data
- **Graceful shutdown**: Handles SIGINT/SIGTERM
- **Job tracking**: API server tracks job status and history

## Usage

### Basic Usage

```bash
# Build
make build

# Run with defaults (10 RPS for 60 seconds)
make run

# Run with custom parameters
./bin/alert-producer -rps 50 -duration 5m -kafka-brokers localhost:9092
```

### Burst Mode

Send a fixed number of alerts immediately:

```bash
make run-burst
# or
./bin/alert-producer -burst 1000 -kafka-brokers localhost:9092
```

### Custom Distributions

```bash
./bin/alert-producer \
  -rps 20 \
  -duration 2m \
  -severity-dist "HIGH:30,MEDIUM:30,LOW:25,CRITICAL:15" \
  -source-dist "api:25,db:20,cache:15,monitor:15,queue:10,worker:5,frontend:5,backend:5" \
  -name-dist "timeout:15,error:15,crash:10,slow:10,memory:10,cpu:10,disk:10,network:10,auth:5,validation:5" \
  -kafka-brokers localhost:9092
```

### Deterministic Mode

Use a seed for reproducible test data:

```bash
make run-deterministic
# or
./bin/alert-producer -seed 42 -rps 10 -duration 30s
```

### Test Mode

Generate varied alerts with one test alert included (LOW/test-source/test-name for client afik-test):

```bash
# Continuous mode: 5 RPS for 30 seconds (includes one test alert + varied alerts)
./bin/alert-producer -test -rps 5 -duration 30s

# Burst mode: send 10 alerts (first is test alert, rest are varied)
./bin/alert-producer -test -burst 10
```

**Note:** Test mode generates varied alerts using the configured distributions, but includes exactly one test alert (LOW/test-source/test-name) at the beginning. This allows you to test your specific rule while also testing other scenarios.

**Testing Multiple Rules Per Client:**
To test the case where a client has multiple rules matching the same alert:
1. Create multiple rules for the same client (e.g., `afik-test`) with the same criteria (LOW/test-source/test-name)
2. Or create rules with different combinations that could match the same alert pattern
3. When an alert matches multiple rules for the same client, the evaluator will:
   - Group them by `client_id` (one message per client)
   - Include all matching `rule_ids[]` in the message
   - The aggregator will dedupe at the `(client_id, alert_id)` level

## Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-kafka-brokers` | `localhost:9092` | Kafka broker addresses (comma-separated) |
| `-topic` | `alerts.new` | Kafka topic name |
| `-rps` | `10.0` | Alerts per second |
| `-duration` | `60s` | Duration to run (e.g., 60s, 5m) |
| `-burst` | `0` | Burst mode: send N alerts immediately (0 = continuous) |
| `-seed` | `0` | Random seed (0 = random) |
| `-severity-dist` | `HIGH:30,MEDIUM:30,LOW:25,CRITICAL:15` | Severity distribution (matches test-data generator) |
| `-source-dist` | `api:25,db:20,cache:15,monitor:15,queue:10,worker:5,frontend:5,backend:5` | Source distribution (matches test-data generator) |
| `-name-dist` | `timeout:15,error:15,crash:10,slow:10,memory:10,cpu:10,disk:10,network:10,auth:5,validation:5` | Name distribution (matches test-data generator) |
| `-mock` | `false` | Use mock producer (no Kafka required, logs alerts instead) |
| `-test` | `false` | Test mode: generate test alerts (LOW/test-source/test-name) matching afik-test rule |

## Prerequisites

- Go 1.22+
- Docker and Docker Compose (for local Kafka)

## Quick Start

### 🚀 Easiest Way (Recommended)

**Start infrastructure first (from root directory):**
```bash
cd ../.. && make setup-infra
```

**Then run the service:**
```bash
make setup-run
# or
./scripts/run-all.sh
```

This automatically:
- ✅ Verifies infrastructure is running
- ✅ Downloads dependencies
- ✅ Builds the service
- ✅ Runs it

See [Getting Started Guide](docs/SETUP_AND_RUN.md) for complete details.

### Manual Steps (Alternative)

1. Start Kafka locally:
```bash
docker compose up -d
# or
make kafka-up
```

2. Wait for Kafka to be ready (about 10-15 seconds), then run the producer:
```bash
make build
make run
```

3. **Test without Kafka** (mock mode):
```bash
./bin/alert-producer --mock -burst 10
```

4. Run the test script to verify everything works:
```bash
./scripts/test-producer.sh
```

## Alert Format

Alerts are published as JSON messages with the following structure:

```json
{
  "alert_id": "uuid",
  "schema_version": 1,
  "event_ts": 1234567890,
  "severity": "HIGH",
  "source": "api",
  "name": "timeout",
  "context": {
    "environment": "prod",
    "region": "us-east-1"
  }
}
```

### Field Descriptions

- **alert_id**: Unique UUID identifier for the alert
- **schema_version**: Schema version (currently 1) for evolution support
- **event_ts**: Unix timestamp when the alert was generated
- **severity**: Enum value - one of `LOW`, `MEDIUM`, or `HIGH`
- **source**: String identifier for the alert source (e.g., "api", "db", "monitor")
- **name**: String identifier for the alert name (e.g., "timeout", "error", "latency")
- **context**: Optional map of additional key-value pairs

Messages are keyed by `alert_id` for even distribution across Kafka partitions.

## Logging

The service uses structured logging (JSON format) with the following log levels:
- **Info**: Normal operation, progress updates, connection status
- **Warn**: Graceful shutdowns, cancellations
- **Error**: Failures, connection errors, publish failures

Sample log output:
```json
{"time":"2024-01-14T16:00:00Z","level":"INFO","msg":"Starting alert-producer","kafka_brokers":"localhost:9092","topic":"alerts.new","rps":10}
{"time":"2024-01-14T16:00:00Z","level":"INFO","msg":"Published first alert (sample)","alert_id":"abc-123","severity":"HIGH","source":"api","name":"timeout"}
```

## Development

```bash
# Download dependencies
make deps

# Run tests
make test

# Build binary
make build

# Clean build artifacts
make clean
```
