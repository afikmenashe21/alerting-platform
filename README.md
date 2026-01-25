# Alerting Platform

[Live Playground (GitHub Pages)](https://afikmenashe21.github.io/alerting-platform/)

A multi-service event-driven platform that evaluates incoming alerts against customer-defined rules and delivers notifications via email, Slack, and webhooks. Built with Go, Kafka, Postgres, and Redis.

## Purpose

Customers (tenants) define rules like: *"If alert has severity=HIGH and source=payments, notify me via email."*

The platform receives a stream of alerts, matches them against all active rules in real-time, deduplicates notifications, and delivers them reliably with at-least-once semantics.

## Architecture
![Alerting Platform Architecture](docs/architecture/alerting-architecture.svg)


### Data Flow

1. **alert-producer** publishes alerts to Kafka topic `alerts.new`
2. **rule-service** exposes a REST API for managing clients, rules, and notification endpoints; publishes `rule.changed` events
3. **rule-updater** consumes `rule.changed`, rebuilds a Redis rule snapshot with inverted indexes
4. **evaluator** consumes `alerts.new`, matches against rules using in-memory indexes (warm-started from Redis), publishes one `alerts.matched` message per matching client
5. **aggregator** consumes `alerts.matched`, performs idempotent insert into Postgres (dedup boundary: unique `client_id + alert_id`), publishes `notifications.ready`
6. **sender** consumes `notifications.ready`, delivers via the configured endpoint (email/Slack/webhook), updates status to `SENT`

## Services

| Service | Role | Port | Kafka Topics |
|---------|------|------|--------------|
| **rule-service** | REST API for rules, clients, endpoints | 8081 | Produces: `rule.changed` |
| **rule-updater** | Maintains Redis rule snapshot | - | Consumes: `rule.changed` |
| **evaluator** | Matches alerts to rules | - | Consumes: `alerts.new`, Produces: `alerts.matched` |
| **aggregator** | Deduplicates notifications | - | Consumes: `alerts.matched`, Produces: `notifications.ready` |
| **sender** | Delivers notifications | - | Consumes: `notifications.ready` |
| **alert-producer** | Generates/publishes alerts (test + API) | 8082 | Produces: `alerts.new` |
| **metrics-service** | Exposes pipeline metrics | 8083 | - |

## Key Design Decisions

- **Idempotency boundary**: Deduplication is enforced at the aggregator via a Postgres unique constraint on `(client_id, alert_id)`, not in-memory. This survives crashes and redeliveries.
- **Rule distribution**: Rules are indexed in Redis as a snapshot; evaluator loads them into memory on startup and refreshes on version change. No full DB scan per alert.
- **Fast matching**: Three inverted indexes (`bySeverity`, `bySource`, `byName`) with set intersection starting from the smallest candidate set.
- **Partitioning**: `alerts.new` keyed by `alert_id` (even distribution), `alerts.matched` keyed by `client_id` (tenant locality).
- **At-least-once delivery**: All Kafka consumers commit offsets after durable progress. Duplicates are safe due to the idempotency boundary.

## Performance

Measured on a single `t3.small` (2 vCPU, 2 GB RAM) with 6 Kafka partitions:

| Component | Throughput | Latency | Notes |
|-----------|-----------|---------|-------|
| Alert Producer | ~780 alerts/s | - | Kafka write limit |
| Evaluator (2 instances) | ~320 alerts/s | 3-160 ms | Depends on matches (see below) |
| Aggregator (2 instances) | ~100 notifications/s | 5.0 ms | DB insert bound |
| Sender | ~120 notifications/s | - | Rate limited |
| **End-to-end pipeline** | **~320 alerts/s** | - | Evaluator is bottleneck |

### Evaluator Latency

Evaluator latency depends on **how many clients match each alert**:

| Scenario | Matching Clients | Latency | Notes |
|----------|-----------------|---------|-------|
| Typical production | 1-10 clients | ~3-5 ms | Rule matching is fast |
| Moderate fan-out | 10-50 clients | ~15-80 ms | Kafka publishing dominates |
| Worst-case (test data) | 99 clients | ~160 ms | 99 Kafka messages per alert |

The evaluator publishes **one Kafka message per matching client**. With test data where all clients have identical rules, every alert matches all clients, causing high fan-out. In production with diverse rules, most alerts match only a few clients.

**Scaling**: Horizontal via Kafka partitions + ECS task count. See [Performance Scaling Guide](docs/deployment/PERFORMANCE_SCALING.md) for capacity planning up to 5,000 alerts/s.

**Current bottlenecks**: Single Kafka broker (~800/s write limit), memory constraint on t3.small (~38 MB free after all services).

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.22+ |
| Messaging | Apache Kafka (segmentio/kafka-go) |
| Database | PostgreSQL 15 |
| Cache | Redis 7 |
| Serialization | Protocol Buffers |
| Infrastructure | Docker Compose (local), Terraform + AWS ECS (production) |
| CI/CD | GitHub Actions |
| UI | React + Vite |

## Quick Start

```bash
# One command to start everything locally:
make run-all
```

This starts infrastructure (Postgres, Kafka, Redis, Zookeeper), runs migrations, and launches all services.

**Step by step:**

```bash
make setup-infra      # Start infrastructure containers
make run-migrations   # Run database migrations
make create-topics    # Create Kafka topics
make run-all          # Start all services
```

See [docs/guides/SETUP.md](docs/guides/SETUP.md) for the complete local setup guide.

## Project Structure

```
alerting-platform/
├── services/              # Microservices (each with Dockerfile, Makefile, go.mod)
│   ├── rule-service/      # REST API
│   ├── rule-updater/      # Redis snapshot builder
│   ├── evaluator/         # Alert-to-rule matcher
│   ├── aggregator/        # Notification deduplicator
│   ├── sender/            # Notification delivery
│   ├── alert-producer/    # Alert generator (test + API)
│   └── metrics-service/   # Pipeline metrics API
├── proto/                 # Protobuf definitions (alerts, rules, notifications)
├── pkg/                   # Shared Go packages (kafka, proto, metrics, shared)
├── terraform/             # AWS infrastructure (VPC, ECS, RDS, Redis, Kafka)
├── scripts/               # Infrastructure, deployment, migration, test scripts
├── rule-service-ui/       # React frontend for rule management
├── docs/                  # Architecture, deployment, and feature documentation
├── migrations/            # Database schema (init-schema.sql)
├── docker-compose.yml     # Local infrastructure
└── Makefile               # Root-level orchestration commands
```

## Production Deployment

Deployed on AWS ECS with Terraform:

```bash
cd terraform
cp terraform.tfvars.example terraform.tfvars  # Configure
terraform init && terraform apply             # Deploy infrastructure
./scripts/deployment/build-and-push.sh        # Build and push Docker images
./scripts/deployment/update-services.sh       # Update ECS services
```

See [docs/deployment/](docs/deployment/) for complete guides:
- [Production Deployment](docs/deployment/PRODUCTION_DEPLOYMENT.md) — full production guide
- [Prerequisites](docs/deployment/PREREQUISITES.md) — required tools and credentials
- [Performance Scaling](docs/deployment/PERFORMANCE_SCALING.md) — load test results and scaling strategies
- [Ultra Low Cost](docs/deployment/ULTRA_LOW_COST.md) — deploy for ~$15/month

## Documentation

- **[docs/guides/SETUP.md](docs/guides/SETUP.md)** — complete local development setup
- **[docs/deployment/](docs/deployment/)** — production deployment guides
- **[docs/architecture/](docs/architecture/)** — infrastructure and protobuf documentation
- **[docs/features/](docs/features/)** — wildcard rules design and usage
- **[proto/README.md](proto/README.md)** — protobuf schema and code generation

## Make Targets

```bash
make help              # Show all available targets
make setup-infra       # Start infrastructure (Postgres, Kafka, Redis, Zookeeper)
make run-migrations    # Run all database migrations
make create-topics     # Create Kafka topics
make run-all           # Run all services
make run-all-bg        # Run all services in background
make stop-all          # Stop everything
make proto-generate    # Generate Go code from .proto files
make generate-test-data # Generate test data (clients, rules, endpoints)
```

## Test Data Generation

Generate test data for load testing:

```bash
# Local: uses localhost Postgres
make generate-test-data

# Remote: specify the database DSN
POSTGRES_DSN="postgres://user:pass@host:5432/db?sslmode=require" make generate-test-data
```

**Default test data**: 100 clients × 288 rules each = 28,800 rules + 57,600 endpoints

Note: With identical rules across all clients, every alert matches all clients (worst-case fan-out). For realistic testing, use fewer clients or vary the rule patterns per client.
