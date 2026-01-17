# Complete Setup Guide

## Overview

This platform uses **centralized infrastructure management**. All services share the same Postgres, Kafka, Redis, Zookeeper, and MailHog instances.

## Initial Setup (One-Time)

### 1. Start All Infrastructure

```bash
make setup-infra
```

This starts:
- Postgres (port 5432) - `alerting-platform-postgres`
- Zookeeper (port 2181) - `alerting-platform-zookeeper`
- Kafka (port 9092) - `alerting-platform-kafka`
- Redis (port 6379) - `alerting-platform-redis`
- MailHog (ports 1025, 8025) - `alerting-platform-mailhog` (SMTP server for email testing)

### 2. Run All Migrations

```bash
make run-migrations
```

This runs **all** migrations from **all** services in the correct order:
- rule-service migrations (000001-000005): clients, rules, endpoints
- aggregator migrations (000006+): notifications

### 3. Create Kafka Topics

```bash
make create-topics
```

This creates all Kafka topics:
- `alerts.new` (3 partitions)
- `rule.changed` (3 partitions)
- `alerts.matched` (3 partitions)
- `notifications.ready` (3 partitions)

## Running Services

After infrastructure is set up, you can run any service:

```bash
# Rule service (HTTP API)
cd rule-service && make run-all

# Rule updater (consumes rule.changed, updates Redis)
cd rule-updater && make run-all

# Evaluator (matches alerts against rules)
cd evaluator && make run-all

# Aggregator (dedupes notifications)
cd aggregator && make run-all

# Sender (sends notifications)
cd sender && make run-all
```

Each service will:
1. ✅ Check Go installation
2. ✅ Verify centralized infrastructure is running
3. ✅ Download Go dependencies
4. ✅ Build the service
5. ✅ Run the service

**Services will NOT:**
- ❌ Start/stop infrastructure
- ❌ Run migrations
- ❌ Create Kafka topics

## Verification

### Check Infrastructure Status

```bash
make verify-deps
```

### Check Migration Status

```bash
make migration-status
```

### Check Migration Consistency

```bash
make check-migrations
```

## Troubleshooting

### Infrastructure Not Running

```bash
# Check what's running
docker ps

# Check logs
docker compose logs

# Restart everything
docker compose down
make setup-infra
```

### Port Conflicts

```bash
# Check what's using ports
lsof -i :5432  # Postgres
lsof -i :9092  # Kafka
lsof -i :6379  # Redis
lsof -i :1025  # MailHog SMTP
lsof -i :8025  # MailHog Web UI
```

### Migration Issues

```bash
# Check status
make migration-status

# Re-run migrations (idempotent)
make run-migrations
```

## Service Dependencies

Each service requires different infrastructure:

| Service | Postgres | Kafka | Redis | Notes |
|---------|----------|-------|-------|-------|
| rule-service | ✅ | ✅ | ❌ | HTTP API, publishes rule.changed |
| rule-updater | ✅ | ✅ | ✅ | Consumes rule.changed, updates Redis |
| evaluator | ❌ | ✅ | ✅ | Matches alerts, needs Redis snapshot |
| aggregator | ✅ | ✅ | ❌ | Dedupes notifications |
| sender | ✅ | ✅ | ❌ | Sends notifications |

All services use the **same** Postgres database (`alerting`) and **same** Kafka cluster.

## Workflow

### First Time Setup

```bash
# 1. Start infrastructure
make setup-infra

# 2. Run migrations
make run-migrations

# 3. Create topics
make create-topics

# 4. Start services (in order)
cd rule-service && make run-all      # Terminal 1
cd rule-updater && make run-all      # Terminal 2
cd evaluator && make run-all         # Terminal 3
cd aggregator && make run-all        # Terminal 4
cd sender && make run-all            # Terminal 5
```

### Daily Development

```bash
# Verify infrastructure is running
make verify-deps

# Run your service
cd <service> && make run-all
```

## Key Principles

1. **Infrastructure is centralized** - One Postgres, one Kafka, one Redis
2. **Services verify, don't manage** - Services check dependencies exist but don't start them
3. **Migrations are centralized** - One script runs all migrations from all services
4. **Topics are centralized** - One script creates all Kafka topics
5. **Container names are consistent** - All use `alerting-platform-*` prefix

See `INFRASTRUCTURE.md` for detailed documentation.
