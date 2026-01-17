# Centralized Infrastructure Management

## Overview

This platform uses **centralized infrastructure management** to avoid conflicts and ensure consistency across all services. Services should **NOT** manage infrastructure themselves - they should only verify dependencies exist.

## Architecture

All services share the same infrastructure:
- **Postgres**: Shared database `alerting` (all services use the same DB)
- **Kafka**: Shared message broker (all topics on same cluster)
- **Zookeeper**: Required for Kafka
- **Redis**: Shared for rule snapshots
- **MailHog**: Local SMTP server for email testing (SMTP on port 1025, Web UI on port 8025)

## Container Names

All infrastructure uses consistent container names:
- `alerting-platform-postgres` - Postgres database
- `alerting-platform-kafka` - Kafka broker
- `alerting-platform-zookeeper` - Zookeeper
- `alerting-platform-redis` - Redis
- `alerting-platform-mailhog` - MailHog SMTP server
- `alerting-platform-kafka-ui` - Kafka UI (optional, for debugging)

## Quick Start

### 1. Start All Infrastructure

```bash
make setup-infra
```

This starts Postgres, Zookeeper, Kafka, Redis, and MailHog with proper configuration.

### 2. Verify Dependencies

```bash
make verify-deps
```

This verifies all services are running and accessible.

### 3. Run Migrations

```bash
make run-migrations
```

This runs **all** migrations from **all** services in the correct order. Services should NOT run migrations themselves.

### 4. Create Kafka Topics

```bash
make create-topics
```

This creates all Kafka topics used by the platform.

## Service Guidelines

### What Services SHOULD Do

1. **Verify dependencies exist** before starting:
   ```bash
   # In service's run script
   ../../scripts/verify-dependencies.sh || exit 1
   ```

2. **Connect to shared infrastructure**:
   - Postgres: `postgres://postgres:postgres@127.0.0.1:5432/alerting?sslmode=disable`
   - Kafka: `localhost:9092`
   - Redis: `localhost:6379`
   - MailHog SMTP: `localhost:1025` (for email testing)
   - MailHog Web UI: `http://localhost:8025` (to view captured emails)

3. **Handle connection failures gracefully** with clear error messages

### What Services SHOULD NOT Do

1. ❌ **Start/stop infrastructure** (Postgres, Kafka, Redis, Zookeeper, MailHog)
2. ❌ **Create/drop databases or tables** (use centralized migrations)
3. ❌ **Create Kafka topics** (use centralized topic creation)
4. ❌ **Manage docker-compose for infrastructure** (use root docker-compose.yml)

## Migration Strategy

Migrations are **centralized** and run from the root:

```bash
make run-migrations
```

This script:
1. Collects all migration files from all services
2. Sorts them by version number
3. Runs them in order against the shared database
4. Verifies tables were created

Services should **NOT** run migrations in their own `run-all.sh` scripts.

## Kafka Topics

All topics are created centrally:

```bash
make create-topics
```

Topics created:
- `alerts.new` (3 partitions)
- `rule.changed` (3 partitions)
- `alerts.matched` (3 partitions)
- `notifications.ready` (3 partitions)

## Troubleshooting

### Infrastructure Not Starting

```bash
# Check what's running
docker ps

# Check logs
docker compose logs postgres
docker compose logs kafka
docker compose logs redis

# Restart everything
docker compose down
make setup-infra
```

### Port Conflicts

If you see port conflicts, check what's using the ports:

```bash
# Check port 5432 (Postgres)
lsof -i :5432

# Check port 9092 (Kafka)
lsof -i :9092

# Check port 6379 (Redis)
lsof -i :6379

# Check port 1025 (MailHog SMTP)
lsof -i :1025

# Check port 8025 (MailHog Web UI)
lsof -i :8025
```

### Migration Issues

```bash
# Check migration status
make migration-status

# Check migration consistency
make check-migrations

# Re-run migrations (idempotent)
make run-migrations
```

## Service-Specific docker-compose.yml

Services may still have their own `docker-compose.yml` for:
- Service-specific test containers
- Development-only services
- Local overrides

But they should **NOT** define shared infrastructure (Postgres, Kafka, Redis, Zookeeper, MailHog).

## Benefits

1. **No port conflicts** - Single instance of each service
2. **Consistent state** - All services see the same data
3. **Easier debugging** - One place to check infrastructure
4. **Faster startup** - Infrastructure starts once, not per-service
5. **Centralized migrations** - No confusion about which service runs which migration
