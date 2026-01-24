# Alert Producer

Generates synthetic alerts and publishes them to Kafka for testing the alerting pipeline. Supports both a CLI tool and an HTTP API server for UI integration.

## Role in Pipeline

```
[alert-producer] → alerts.new (Kafka) → evaluator → ...
```

The alert-producer is a **test/development tool**, not a production service. It simulates upstream alert sources with configurable distributions and rates.

## Modes

### CLI Mode (default binary)

Generate alerts from the command line:

```bash
# Continuous: 10 alerts/s for 60 seconds
./bin/alert-producer -rps 10 -duration 60s

# Burst: send 1000 alerts immediately
./bin/alert-producer -burst 1000

# Deterministic: reproducible with seed
./bin/alert-producer -seed 42 -rps 10 -duration 30s

# Mock mode: no Kafka required, logs alerts
./bin/alert-producer -mock -burst 10
```

### API Server Mode (for UI)

HTTP API that the React UI uses to trigger alert generation:

```bash
./bin/alert-producer-api -http-port 8082
```

Endpoints:
- `POST /api/v1/alerts/generate` — start a generation job
- `GET /api/v1/alerts/jobs` — list job history
- `GET /api/v1/alerts/jobs/:id` — get job status
- `GET /health` — health check

## Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `-kafka-brokers` | `localhost:9092` | Kafka broker addresses |
| `-topic` | `alerts.new` | Output Kafka topic |
| `-rps` | `10` | Alerts per second (continuous mode) |
| `-duration` | `60s` | Duration to run |
| `-burst` | `0` | Burst mode: send N alerts immediately |
| `-seed` | `0` | Random seed (0 = random) |
| `-mock` | `false` | Use mock producer (no Kafka) |
| `-test` | `false` | Include a test alert matching `afik-test` rule |
| `-severity-dist` | `HIGH:30,MEDIUM:30,LOW:25,CRITICAL:15` | Severity distribution |
| `-source-dist` | `api:25,db:20,cache:15,...` | Source distribution |
| `-name-dist` | `timeout:15,error:15,crash:10,...` | Name distribution |

## Alert Format

Published as protobuf (wire) / JSON-equivalent:

```json
{
  "alert_id": "uuid-v4",
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

Messages are keyed by `alert_id` for even distribution across Kafka partitions.

## Running

```bash
# From project root: start infrastructure first
make setup-infra

# CLI mode
cd services/alert-producer
make build
make run            # continuous (10 RPS, 60s)
make run-burst      # burst (1000 alerts)

# API server mode (for UI)
make run-api
```

## Testing

```bash
make test
```

## Key Properties

- **Configurable distributions**: Match test-data generator values so alerts hit existing rules
- **Rate control**: Continuous (RPS) or burst (fixed count) modes
- **Partitioning**: Keyed by `alert_id` for even Kafka partition distribution
- **Graceful shutdown**: Handles SIGINT/SIGTERM, reports final stats
