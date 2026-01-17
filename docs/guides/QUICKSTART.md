# Quick Start Guide

## Centralized Infrastructure Setup

This platform uses **centralized infrastructure management**. All services share the same Postgres, Kafka, Redis, and Zookeeper instances.

### Step 1: Start Infrastructure

```bash
make setup-infra
```

This starts:
- Postgres (port 5432)
- Zookeeper (port 2181)
- Kafka (port 9092)
- Redis (port 6379)

### Step 2: Verify Dependencies

```bash
make verify-deps
```

This verifies all infrastructure is running and accessible.

### Step 3: Run Migrations

```bash
make run-migrations
```

This runs **all** migrations from **all** services in the correct order. The script:
- Collects migrations from all services
- Sorts them by version number
- Runs them against the shared database
- Verifies tables were created

### Step 4: Create Kafka Topics

```bash
make create-topics
```

This creates all Kafka topics:
- `alerts.new`
- `rule.changed`
- `alerts.matched`
- `notifications.ready`

### Step 5: Run All Services

```bash
make run-all
```

This starts all services:
- **rule-service** - HTTP API (port 8081)
- **rule-updater** - Rule snapshot updater
- **evaluator** - Alert matcher
- **aggregator** - Notification deduplicator
- **sender** - Notification sender
- **alert-producer** - Test alert generator

Services will start in separate terminal windows (or background if using `make run-all-bg`).

### Step 5: Run Services

Now you can run individual services. Each service will:
- Verify dependencies exist (but won't start them)
- Connect to shared infrastructure
- Start the service

Example:
```bash
cd rule-service
make run-all
```

## Service Guidelines

### ✅ Services SHOULD:
- Verify dependencies before starting
- Connect to shared infrastructure
- Handle connection failures gracefully

### ❌ Services SHOULD NOT:
- Start/stop infrastructure
- Create/drop databases or tables
- Create Kafka topics
- Manage docker-compose for infrastructure

## Troubleshooting

### Infrastructure Not Running

```bash
# Check status
docker ps

# Check logs
docker compose logs

# Restart
docker compose down
make setup-infra
```

### Port Conflicts

If ports are in use, check what's using them:
```bash
lsof -i :5432  # Postgres
lsof -i :9092  # Kafka
lsof -i :6379  # Redis
```

### Migration Issues

```bash
# Check status
make migration-status

# Check consistency
make check-migrations

# Re-run (idempotent)
make run-migrations
```

## Full Documentation

See `INFRASTRUCTURE.md` for complete documentation on the centralized infrastructure system.
