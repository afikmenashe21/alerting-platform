# Migration to Centralized Infrastructure

## What Changed

All service `run-all.sh` scripts have been updated to use **centralized infrastructure management**.

### Before (Old Approach)
- Each service managed its own infrastructure
- Services started Postgres, Kafka, Redis independently
- Services ran their own migrations
- Services created their own Kafka topics
- Result: Port conflicts, inconsistent state, duplication

### After (New Approach)
- ✅ Centralized infrastructure (`docker-compose.yml` at root)
- ✅ Services only verify dependencies exist
- ✅ Centralized migration runner
- ✅ Centralized Kafka topic creation
- ✅ Consistent container names

## Updated Services

All these services now use centralized infrastructure:

1. **rule-service** - Verifies Postgres + Kafka
2. **rule-updater** - Verifies Postgres + Kafka + Redis
3. **evaluator** - Verifies Kafka + Redis
4. **aggregator** - Verifies Postgres + Kafka
5. **sender** - Verifies Postgres + Kafka

## New Workflow

### Initial Setup (One Time)

```bash
# 1. Start all infrastructure
make setup-infra

# 2. Run all migrations (from all services)
make run-migrations

# 3. Create all Kafka topics
make create-topics
```

### Running Services

```bash
# Each service now just verifies and runs
cd rule-service && make run-all
cd aggregator && make run-all
# etc.
```

## What Services Do Now

Each service `run-all.sh` script now:

1. ✅ Checks Go installation
2. ✅ Verifies centralized infrastructure (calls `scripts/infrastructure/verify-dependencies.sh`)
3. ✅ Downloads Go dependencies
4. ✅ Builds the service
5. ✅ Runs the service

**Removed from all services:**
- ❌ Infrastructure startup (Postgres, Kafka, Redis, Zookeeper)
- ❌ Migration running
- ❌ Kafka topic creation
- ❌ Docker compose management

## Container Names

All infrastructure now uses consistent names:
- `alerting-platform-postgres`
- `alerting-platform-kafka`
- `alerting-platform-zookeeper`
- `alerting-platform-redis`

Old service-specific containers (e.g., `rule-service-postgres`) should be stopped and removed.

## Migration Steps

If you have old containers running:

```bash
# Stop all old containers
docker stop $(docker ps -q --filter "name=postgres") 2>/dev/null || true
docker stop $(docker ps -q --filter "name=kafka") 2>/dev/null || true
docker stop $(docker ps -q --filter "name=redis") 2>/dev/null || true

# Start centralized infrastructure
make setup-infra

# Run migrations
make run-migrations

# Create topics
make create-topics
```

## Benefits

1. **No port conflicts** - Single instance of each service
2. **Consistent state** - All services see the same data
3. **Easier debugging** - One place to check infrastructure
4. **Faster startup** - Infrastructure starts once
5. **Centralized migrations** - No confusion about which service runs which migration

## Documentation

- `INFRASTRUCTURE.md` - Complete infrastructure guide
- `SETUP.md` - Setup instructions
- `QUICKSTART.md` - Quick start guide
- `scripts/infrastructure/verify-dependencies.sh` - Dependency verification
- `scripts/migrations/run-migrations.sh` - Centralized migration runner
- `scripts/infrastructure/create-kafka-topics.sh` - Centralized topic creation
