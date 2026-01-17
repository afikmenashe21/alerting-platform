# Aggregator Service

The aggregator service consumes matched alerts from Kafka, persists them idempotently to PostgreSQL, and emits notification ready events for newly created notifications.

## Purpose

The aggregator provides the **idempotency boundary** for the alerting platform. It ensures that:
- No duplicate notifications are created for the same `(client_id, alert_id)` pair
- Kafka redeliveries are handled gracefully via database unique constraint
- Only newly created notifications trigger `notifications.ready` events

## Architecture

```
alerts.matched (Kafka) → Aggregator → PostgreSQL (notifications table) → notifications.ready (Kafka)
```

### Design Pattern

The service uses a **Processor Pattern** to separate business logic from initialization:
- `cmd/aggregator/main.go`: CLI entry point and initialization
- `internal/processor/`: Business logic for notification aggregation
- `internal/database/`: Data access layer with idempotent inserts
- `internal/consumer/`: Kafka consumer abstraction
- `internal/producer/`: Kafka producer abstraction

This modular design provides:
- Clear separation of concerns
- Testable business logic
- Easy extensibility

## Key Components

1. **Consumer**: Reads `alerts.matched` messages from Kafka
2. **Database**: Idempotent insert into `notifications` table with unique constraint on `(client_id, alert_id)`
3. **Producer**: Publishes `notifications.ready` events only for newly created notifications

### Idempotency Mechanism

The service uses PostgreSQL's `INSERT ... ON CONFLICT DO NOTHING RETURNING` pattern:
- If a notification with the same `(client_id, alert_id)` already exists, the insert is skipped
- Only new notifications return a `notification_id`, which triggers a `notifications.ready` event
- This ensures at-least-once processing without duplicates

## Prerequisites

- Go 1.22+
- PostgreSQL 15+
- Kafka (via Docker Compose)
- `golang-migrate` tool for database migrations

## Installation

1. Install dependencies:
```bash
make deps
```

2. Install migration tool (if not already installed):
```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

## Setup

Infrastructure (Kafka, Postgres, Redis) is managed centrally at the root level.

1. Start centralized infrastructure (from project root):
```bash
cd ../..
make setup-infra
```

This will:
- Start Kafka, Postgres, Redis, and other shared services
- Run all database migrations (including aggregator migrations)
- Create required Kafka topics

**Note**: The aggregator service uses centralized infrastructure. Individual service setup commands have been removed in favor of centralized management.

## Running

### Quick Start (Recommended)

**Prerequisites**: Ensure centralized infrastructure is running (from project root: `make setup-infra`)

Run the aggregator service:

```bash
make run-all
```

This command will:
1. ✅ Check Go installation (1.22+)
2. ✅ Verify centralized infrastructure is available
3. ✅ Download Go dependencies
4. ✅ Build the service
5. ✅ Run the service

### Manual Steps

#### Build and run:
```bash
make run
```

#### Run with custom configuration:
```bash
make run ARGS="-kafka-brokers localhost:9092 -postgres-dsn postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable"
```

#### Default configuration:
- Kafka brokers: `localhost:9092`
- Alerts matched topic: `alerts.matched`
- Notifications ready topic: `notifications.ready`
- Consumer group: `aggregator-group`
- Postgres DSN: `postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable`

## Database Schema

The `notifications` table has:
- `notification_id` (UUID, primary key)
- `client_id` (VARCHAR, part of unique constraint)
- `alert_id` (VARCHAR, part of unique constraint)
- `severity`, `source`, `name` (alert fields)
- `context` (JSONB, optional alert context)
- `rule_ids` (TEXT[], array of matching rule IDs)
- `status` (VARCHAR, default 'RECEIVED')
- `created_at`, `updated_at` (timestamps)
- **Unique constraint**: `(client_id, alert_id)` - ensures idempotency

## Migration Strategy

This service uses **migration version 000006** (after rule-service migrations 000001-000005).

**Important**: When creating new migrations:
1. Check current highest version: `cd .. && make check-migrations`
2. Use next sequential number (e.g., if highest is 000006, use 000007)
3. See `../migrations/MIGRATION_STRATEGY.md` for full strategy

## Event Formats

### Input: `alerts.matched`
```json
{
  "alert_id": "alert-123",
  "schema_version": 1,
  "event_ts": 1234567890,
  "severity": "HIGH",
  "source": "monitoring",
  "name": "cpu_high",
  "context": {"host": "server1"},
  "client_id": "client-456",
  "rule_ids": ["rule-789", "rule-790"]
}
```

### Output: `notifications.ready`
```json
{
  "notification_id": "550e8400-e29b-41d4-a716-446655440000",
  "client_id": "client-456",
  "alert_id": "alert-123",
  "schema_version": 1
}
```

## Processing Flow

1. Read `alerts.matched` message from Kafka
2. Attempt idempotent insert into `notifications` table
3. If new notification created (notification_id returned):
   - Publish `notifications.ready` event
4. Commit Kafka offset (only after successful DB operation)

## Offset Commit Strategy

Offsets are committed only after:
- Successful database insert (or confirmed duplicate)
- Successful publish of `notifications.ready` (if applicable)

This ensures at-least-once semantics: if the service crashes before committing, Kafka will redeliver the message, and the idempotent insert will prevent duplicates.

## Testing

Run tests:
```bash
make test
```

## Development

### Create a new migration:
```bash
make migrate-create NAME=add_index_to_notifications
```

**Note**: Check `../migrations/MIGRATION_STRATEGY.md` first to determine the correct version number.

### Run migrations down:
```bash
make migrate-down
```

### View logs:
```bash
make logs
```

### Check service status:
```bash
make status
```

## Troubleshooting

### Database connection errors
- Ensure Postgres is running: `docker ps | grep alerting-platform-postgres`
- Check Postgres logs: `docker logs alerting-platform-postgres`
- Verify DSN format and credentials
- Start infrastructure from root: `cd ../.. && make setup-infra`

### Kafka connection errors
- Ensure Kafka is running: `docker ps | grep alerting-platform-kafka`
- Check Kafka logs: `docker logs alerting-platform-kafka`
- Verify topics exist (topics are auto-created, or use root: `cd ../.. && make create-topics`)
- Start infrastructure from root: `cd ../.. && make setup-infra`

### Migration errors
- Ensure `migrate` tool is installed
- Check Postgres is accessible
- Verify migration files are in `migrations/` directory
- **Check migration consistency**: `cd .. && make check-migrations`

### Migration version conflicts
If you see "no migration found for version X":
1. Check migration consistency: `cd .. && make check-migrations`
2. Verify all services use sequential version numbers
3. See `../migrations/MIGRATION_STRATEGY.md` for versioning strategy

## Viewing Database Records

### Quick queries:
```bash
# Show recent notifications (last 20)
make db-query

# Show notification counts by status
make db-count

# Show all notifications
make db-all

# Show only RECEIVED notifications (waiting to be sent)
make db-received
```

### Interactive PostgreSQL shell:
```bash
make db-psql
```

Then you can run SQL queries directly:
```sql
-- Count notifications by status
SELECT status, COUNT(*) FROM notifications GROUP BY status;

-- Find notifications for a specific client
SELECT * FROM notifications WHERE client_id = 'client-1' ORDER BY created_at DESC;

-- Find notifications for a specific alert
SELECT * FROM notifications WHERE alert_id = 'alert-123';

-- View notifications with their rule IDs
SELECT notification_id, client_id, alert_id, rule_ids, status, created_at 
FROM notifications 
ORDER BY created_at DESC 
LIMIT 10;
```

### Direct docker command:
```bash
# Query directly (using centralized container name)
docker exec -it alerting-platform-postgres psql -U postgres -d alerting -c "SELECT * FROM notifications;"

# Or open interactive shell
docker exec -it alerting-platform-postgres psql -U postgres -d alerting
```

## Cleanup

Stop infrastructure (from project root):
```bash
cd ../..
make stop-infra
```

Clean build artifacts:
```bash
make clean
```
