# Rule Service

Control-plane API service for managing clients and alerting rules. Persists rules to Postgres and publishes `rule.changed` events to Kafka for data-plane services to consume.

## Overview

The rule-service provides a REST API for:
- **Clients**: Managing tenant/organization records
- **Rules**: CRUD operations for alert matching rules
- **Endpoints**: Managing notification endpoints for rules (email, webhook, slack)

When rules are created, updated, deleted, or disabled, the service publishes `rule.changed` events to Kafka, which are consumed by the `rule-updater` service to rebuild rule snapshots in Redis.

## Features

- RESTful HTTP API for clients, rules, and endpoints
- Postgres persistence with migrations
- Optimistic locking (version-based) for rule updates
- Kafka event publishing (`rule.changed` topic)
- Graceful shutdown handling
- Modular architecture with router and handler patterns

## Documentation

- **[Architecture](docs/ARCHITECTURE.md)** - Service architecture and design patterns

## Database Schema

The service uses three main tables with proper relationships:

### Clients Table
- `client_id` (VARCHAR, PRIMARY KEY)
- `name` (VARCHAR)
- `created_at`, `updated_at` (TIMESTAMP)

### Rules Table
- `rule_id` (UUID, PRIMARY KEY)
- `client_id` (VARCHAR, FOREIGN KEY → clients)
- `severity` (VARCHAR, enum: LOW/MEDIUM/HIGH/CRITICAL)
- `source` (VARCHAR)
- `name` (VARCHAR)
- `enabled` (BOOLEAN)
- `version` (INTEGER, for optimistic locking)
- `created_at`, `updated_at` (TIMESTAMP)
- Unique constraint: `(client_id, severity, source, name)`

### Endpoints Table
- `endpoint_id` (UUID, PRIMARY KEY)
- `rule_id` (UUID, FOREIGN KEY → rules, ON DELETE CASCADE)
- `type` (VARCHAR, enum: email/webhook/slack)
- `value` (VARCHAR, e.g., email address, URL)
- `enabled` (BOOLEAN)
- `created_at`, `updated_at` (TIMESTAMP)
- Unique constraint: `(rule_id, type, value)`

### Relationships
- **Client → Rules**: One-to-many (a client has many rules)
- **Rule → Endpoints**: One-to-many (a rule can have many endpoints)

## Rule Model (MVP)

Rules use exact-match criteria:
- `severity`: Enum (LOW, MEDIUM, HIGH, CRITICAL)
- `source`: String (e.g., "api", "db", "monitor")
- `name`: String (e.g., "timeout", "error", "latency")

Each rule belongs to a `client_id` and can have multiple endpoints (email, webhook, slack) for notifications.

## API Endpoints

### Clients

- `POST /api/v1/clients` - Create a client
- `GET /api/v1/clients?client_id=<id>` - Get a client by ID
- `GET /api/v1/clients` - List all clients

### Rules

- `POST /api/v1/rules` - Create a rule
- `GET /api/v1/rules?rule_id=<id>` - Get a rule by ID
- `GET /api/v1/rules?client_id=<id>` - List rules for a client
- `GET /api/v1/rules` - List all rules
- `PUT /api/v1/rules/update?rule_id=<id>` - Update a rule (requires version for optimistic locking)
- `POST /api/v1/rules/toggle?rule_id=<id>` - Toggle rule enabled/disabled (requires version)
- `DELETE /api/v1/rules/delete?rule_id=<id>` - Delete a rule

### Endpoints

- `POST /api/v1/endpoints` - Create an endpoint for a rule
- `GET /api/v1/endpoints?endpoint_id=<id>` - Get an endpoint by ID
- `GET /api/v1/endpoints?rule_id=<id>` - List all endpoints for a rule
- `PUT /api/v1/endpoints/update?endpoint_id=<id>` - Update an endpoint
- `POST /api/v1/endpoints/toggle?endpoint_id=<id>` - Toggle endpoint enabled/disabled
- `DELETE /api/v1/endpoints/delete?endpoint_id=<id>` - Delete an endpoint

### Health

- `GET /health` - Health check endpoint

## Quick Start

### Prerequisites

- Go 1.22+
- Docker and Docker Compose
- `golang-migrate` tool (will be auto-installed if missing)

### One-Command Setup and Run (Recommended)

The easiest way to get started is with a single command that verifies all dependencies, sets up infrastructure, and runs the service:

```bash
make run-all
```

This command will:
1. ✅ Verify Go 1.22+ is installed
2. ✅ Check Docker is installed and running
3. ✅ Install `golang-migrate` tool if missing
4. ✅ Download Go dependencies
5. ✅ Start Postgres, Kafka, and Zookeeper (if not already running)
6. ✅ Wait for services to be ready
7. ✅ Run database migrations
8. ✅ Create Kafka topics
9. ✅ Build the service
10. ✅ Start the HTTP server

The service will be available at `http://localhost:8081` once started.

### Manual Setup (Alternative)

If you prefer to set up manually:

```bash
# Install dependencies
make deps

# Start Postgres and Kafka, run migrations, create topics
make setup

# Run the service
make run-default
```

### Manual Setup

1. Start dependencies:
```bash
docker compose up -d
```

2. Run migrations:
```bash
make migrate-up
```

3. Create Kafka topic:
```bash
make create-topics
```

4. Run the service:
```bash
make run
```

## Configuration

Command-line flags:

- `-http-port`: HTTP server port (default: 8081)
- `-kafka-brokers`: Kafka broker addresses (default: localhost:9092)
- `-rule-changed-topic`: Kafka topic for rule changed events (default: rule.changed)
- `-postgres-dsn`: PostgreSQL connection string (default: postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable)

## Example API Usage

### Create a Client

```bash
curl -X POST http://localhost:8081/api/v1/clients \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "client-1",
    "name": "Acme Corp"
  }'
```

### Create a Rule

```bash
curl -X POST http://localhost:8081/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "client-1",
    "severity": "HIGH",
    "source": "api",
    "name": "timeout"
  }'
```

### Create an Endpoint for a Rule

```bash
curl -X POST http://localhost:8081/api/v1/endpoints \
  -H "Content-Type: application/json" \
  -d '{
    "rule_id": "<rule-id>",
    "type": "email",
    "value": "ops@acme.com"
  }'
```

### List Endpoints for a Rule

```bash
curl http://localhost:8081/api/v1/endpoints?rule_id=<rule-id>
```

### List Rules for a Client

```bash
curl http://localhost:8081/api/v1/rules?client_id=client-1
```

### Update a Rule

```bash
curl -X PUT "http://localhost:8081/api/v1/rules/update?rule_id=<rule-id>" \
  -H "Content-Type: application/json" \
  -d '{
    "severity": "CRITICAL",
    "source": "api",
    "name": "timeout",
    "version": 1
  }'
```

### Update an Endpoint

```bash
curl -X PUT "http://localhost:8081/api/v1/endpoints/update?endpoint_id=<endpoint-id>" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "webhook",
    "value": "https://hooks.example.com/alerts"
  }'
```

## Database Migrations

Migrations are located in `migrations/` directory.

- Run migrations: `make migrate-up`
- Rollback migrations: `make migrate-down`
- Create new migration: `make migrate-create NAME=migration_name`

## Database Queries

- View recent rules: `make db-query`
- View all clients: `make db-clients`
- View all rules: `make db-rules`
- Open PostgreSQL shell: `make db-psql`

### Example Database Queries

```sql
-- List all clients with their rule counts
SELECT c.client_id, c.name, COUNT(r.rule_id) as rule_count
FROM clients c
LEFT JOIN rules r ON c.client_id = r.client_id
GROUP BY c.client_id, c.name;

-- List all rules with their endpoint counts
SELECT r.rule_id, r.client_id, r.severity, r.source, r.name, COUNT(e.endpoint_id) as endpoint_count
FROM rules r
LEFT JOIN endpoints e ON r.rule_id = e.rule_id
GROUP BY r.rule_id, r.client_id, r.severity, r.source, r.name;

-- List all enabled endpoints for a specific rule
SELECT e.endpoint_id, e.type, e.value, e.enabled
FROM endpoints e
WHERE e.rule_id = '<rule-id>' AND e.enabled = TRUE;
```

## Event Publishing

When a rule is created, updated, deleted, or disabled, a `rule.changed` event is published to Kafka with:

```json
{
  "rule_id": "uuid",
  "client_id": "client-1",
  "action": "CREATED|UPDATED|DELETED|DISABLED",
  "version": 1,
  "updated_at": 1234567890,
  "schema_version": 1
}
```

Events are keyed by `rule_id` for partition distribution.

### Event Publishing Validation

The service publishes `rule.changed` events in the following scenarios:

- ✅ **Rule Created**: `POST /api/v1/rules` → `CREATED` event
- ✅ **Rule Updated**: `PUT /api/v1/rules/update` → `UPDATED` event
- ✅ **Rule Disabled**: `POST /api/v1/rules/toggle` (enabled=false) → `DISABLED` event
- ✅ **Rule Enabled**: `POST /api/v1/rules/toggle` (enabled=true) → `UPDATED` event
- ✅ **Rule Deleted**: `DELETE /api/v1/rules/delete` → `DELETED` event

All events are published **after** successful database commits, ensuring data consistency.

### Testing Event Publishing

Test that events are properly published:

```bash
make test-events
```

This will:
1. Create a test client and rule
2. Verify the event is published to Kafka
3. Test update and delete operations
4. Validate event structure

See `docs/EVENT_VALIDATION.md` for detailed validation documentation.

## Project Structure

```
rule-service/
├── cmd/
│   └── rule-service/
│       └── main.go          # Entry point
├── internal/
│   ├── config/              # Configuration
│   ├── database/            # Database operations
│   ├── events/              # Event structures
│   ├── handlers/            # HTTP handlers
│   └── producer/            # Kafka producer
├── migrations/              # Database migrations
├── Makefile
├── docker-compose.yml       # Centralized infrastructure (at root level)
└── README.md
```

## Development

### Running Tests

```bash
make test
```

### Building

```bash
make build
```

The binary will be in `bin/rule-service`.

## Integration

The rule-service integrates with:
- **rule-updater**: Consumes `rule.changed` events to rebuild Redis snapshots
- **evaluator**: Uses rule snapshots from Redis for alert matching

## Notes

- Optimistic locking: Rule updates require the current `version` to prevent concurrent modification conflicts
- Event publishing happens after successful DB commits (MVP approach; outbox pattern can be added later)
- Rule deletion publishes a `DELETED` event with the rule's last known state
- Disabling a rule publishes a `DISABLED` event; re-enabling publishes an `UPDATED` event
