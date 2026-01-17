# Alerting Platform

A multi-service Go application implementing an end-to-end alert notification platform with Kafka, Postgres, and Redis.

## Quick Start

```bash
# 1. Start all infrastructure and run all services (one command!)
make run-all
```

This single command will:
- ✅ Start infrastructure (Postgres, Kafka, Redis, Zookeeper) if not running
- ✅ Run all migrations automatically (including wildcard support migration)
- ✅ Start all services

**Or step by step:**
```bash
# 1. Start all infrastructure
make setup-infra

# 2. Run all migrations
make run-migrations

# 3. Create Kafka topics
make create-topics

# 4. Run all services
make run-all
```

## Services

All services are located in `services/`:

- **rule-service** - HTTP API for managing clients, rules, and endpoints
- **rule-updater** - Consumes `rule.changed` events, updates Redis snapshots
- **evaluator** - Matches alerts against rules, emits `alerts.matched`
- **aggregator** - Deduplicates notifications, emits `notifications.ready`
- **sender** - Sends notifications via email (SMTP), Slack, and webhooks
- **alert-producer** - Generates and publishes test alerts

## Architecture

See `memory-bank/projectbrief.md` for the complete architecture overview.

## Documentation

Documentation is organized in the `docs/` directory:

- **Guides** (`docs/guides/`):
  - `SETUP.md` - Complete setup guide
  - `QUICKSTART.md` - Quick start instructions

- **Architecture** (`docs/architecture/`):
  - `INFRASTRUCTURE.md` - Infrastructure management details
  - `MIGRATION_TO_CENTRALIZED.md` - Migration guide

- **Features** (`docs/features/`):
  - `WILDCARD_RULES_DESIGN.md` - Wildcard rules design documentation
  - `WILDCARD_RULES_USAGE.md` - Wildcard rules usage guide

See `docs/README.md` for complete documentation index.

## Make Targets

```bash
make help              # Show all available targets
make setup-infra        # Start all infrastructure
make verify-deps        # Verify dependencies
make run-migrations     # Run all migrations
make create-topics      # Create Kafka topics
make run-all            # Run all services
make run-all-bg         # Run all services in background
```

## Directory Structure

```
alerting-platform/
├── services/           # All services
│   ├── rule-service/
│   ├── rule-updater/
│   ├── evaluator/
│   ├── aggregator/
│   ├── sender/
│   └── alert-producer/
├── docs/              # Documentation
│   ├── guides/        # Setup and quick start guides
│   ├── architecture/  # Architecture and infrastructure docs
│   └── features/      # Feature-specific documentation
├── scripts/           # Centralized scripts
│   ├── setup-infrastructure.sh
│   ├── verify-dependencies.sh
│   ├── run-migrations.sh
│   ├── create-kafka-topics.sh
│   └── run-all-services.sh
├── migrations/        # Migration strategy documentation
├── memory-bank/       # Project memory bank (design decisions)
├── rule-service-ui/  # React UI for rule-service
├── docker-compose.yml # Centralized infrastructure
└── Makefile          # Root-level commands
```

## Infrastructure

All infrastructure is managed centrally:
- **Postgres** - `alerting-platform-postgres` (port 5432)
- **Kafka** - `alerting-platform-kafka` (port 9092)
- **Zookeeper** - `alerting-platform-zookeeper` (port 2181)
- **Redis** - `alerting-platform-redis` (port 6379)

Services verify dependencies but do NOT manage them.

## Development

Each service has its own `run-all.sh` script that:
1. Verifies Go installation
2. Verifies centralized infrastructure
3. Downloads dependencies
4. Builds the service
5. Runs the service

See individual service READMEs for service-specific details.
