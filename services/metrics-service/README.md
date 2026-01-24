# Metrics Service

HTTP API that exposes real-time pipeline metrics (throughput, latency, error rates) by querying service metrics from the shared metrics package.

## Role in Pipeline

```
All services → pkg/metrics (shared) → [metrics-service] → HTTP API → UI / monitoring
```

The metrics-service provides observability into the pipeline without requiring external monitoring infrastructure. It reads metrics collected by the shared `pkg/metrics` package.

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/services/metrics` | All service metrics |
| `GET` | `/health` | Health check |

### Response Format

```json
{
  "services": {
    "evaluator": {
      "messages_processed": 50000,
      "messages_per_second": 320.5,
      "avg_processing_latency_ns": 3200000,
      "processing_errors": 0
    },
    "aggregator": { ... },
    "sender": { ... }
  }
}
```

## Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `-http-port` | `8083` | HTTP server port |
| `-postgres-dsn` | `postgres://...` | Postgres connection string |
| `-redis-addr` | `localhost:6379` | Redis address |

## Running

```bash
# From project root: start infrastructure first
make setup-infra

# Then run this service
cd services/metrics-service
go run ./cmd/metrics-service
```

The service will be available at `http://localhost:8083`.
