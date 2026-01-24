# Rule Service

REST API for managing clients, alerting rules, and notification endpoints. Publishes `rule.changed` events to Kafka when rules are modified.

## Role in Pipeline

```
HTTP clients (UI, curl) → [rule-service] → Postgres (clients, rules, endpoints)
                                          → rule.changed (Kafka)
```

The rule-service is the **control plane** of the platform. It manages the rule configuration that the data-plane services (evaluator, aggregator, sender) consume.

## API Endpoints

### Clients

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/clients` | Create a client |
| `GET` | `/api/v1/clients` | List all clients |
| `GET` | `/api/v1/clients?client_id=<id>` | Get a client |

### Rules

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/rules` | Create a rule |
| `GET` | `/api/v1/rules` | List all rules |
| `GET` | `/api/v1/rules?client_id=<id>` | List rules for a client |
| `GET` | `/api/v1/rules?rule_id=<id>` | Get a rule |
| `PUT` | `/api/v1/rules/update?rule_id=<id>` | Update a rule (requires `version`) |
| `POST` | `/api/v1/rules/toggle?rule_id=<id>` | Toggle enabled/disabled (requires `version`) |
| `DELETE` | `/api/v1/rules/delete?rule_id=<id>` | Delete a rule |

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/endpoints` | Create an endpoint for a rule |
| `GET` | `/api/v1/endpoints?rule_id=<id>` | List endpoints for a rule |
| `PUT` | `/api/v1/endpoints/update?endpoint_id=<id>` | Update an endpoint |
| `POST` | `/api/v1/endpoints/toggle?endpoint_id=<id>` | Toggle enabled/disabled |
| `DELETE` | `/api/v1/endpoints/delete?endpoint_id=<id>` | Delete an endpoint |

### Health

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |

## Rule Model

Rules match alerts on three fields (exact match or wildcard `*`):

| Field | Type | Values |
|-------|------|--------|
| `severity` | enum | `LOW`, `MEDIUM`, `HIGH`, `CRITICAL`, `*` |
| `source` | string | Any string or `*` |
| `name` | string | Any string or `*` |

Each rule belongs to a `client_id` and can have multiple notification endpoints (email, webhook, slack).

Optimistic locking: updates require the current `version` field to prevent concurrent modification.

## Events

When rules are created, updated, deleted, or toggled, a `rule.changed` event is published:

```json
{
  "rule_id": "uuid",
  "client_id": "client-1",
  "action": "CREATED|UPDATED|DELETED|DISABLED",
  "version": 1,
  "updated_at": 1234567890
}
```

Events are published **after** successful DB commit. Keyed by `rule_id`.

## Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `-http-port` | `8081` | HTTP server port |
| `-kafka-brokers` | `localhost:9092` | Kafka broker addresses |
| `-rule-changed-topic` | `rule.changed` | Kafka topic for rule events |
| `-postgres-dsn` | `postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable` | Postgres connection string |

## Database Schema

```
clients (client_id PK, name)
    ↓ 1:N
rules (rule_id PK, client_id FK, severity, source, name, enabled, version)
    ↓ 1:N
endpoints (endpoint_id PK, rule_id FK CASCADE, type, value, enabled)
```

Unique constraints:
- `rules`: `(client_id, severity, source, name)`
- `endpoints`: `(rule_id, type, value)`

Migrations: `000001` through `000007` in `migrations/`

## Running

```bash
# From project root: start infrastructure first
make setup-infra && make run-migrations

# Then run this service
cd services/rule-service
make run-all
```

The service will be available at `http://localhost:8081`.

## Example Usage

```bash
# Create a client
curl -X POST http://localhost:8081/api/v1/clients \
  -H "Content-Type: application/json" \
  -d '{"client_id": "acme", "name": "Acme Corp"}'

# Create a rule
curl -X POST http://localhost:8081/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{"client_id": "acme", "severity": "HIGH", "source": "api", "name": "timeout"}'

# Add an email endpoint to the rule
curl -X POST http://localhost:8081/api/v1/endpoints \
  -H "Content-Type: application/json" \
  -d '{"rule_id": "<rule-id>", "type": "email", "value": "ops@acme.com"}'
```

## Testing

```bash
make test
```
