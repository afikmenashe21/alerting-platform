# Alerting Platform – Tech Context

## Language & runtime
- Go **1.22+** for all services.

## Messaging
- Kafka (local via docker compose)
- Go client: `github.com/segmentio/kafka-go`
- Topics (MVP):
  - `alerts.new` (producer → evaluator)
  - `rule.changed` (rule-service → rule-updater/evaluator)
  - `alerts.matched` (evaluator → aggregator)
  - `notifications.ready` (aggregator → sender)

### Kafka semantics we rely on
- Consumer groups: a partition is processed by only one consumer instance at a time.
- At-least-once delivery: duplicates possible after crashes/rebalances.
- Offsets committed by consumer; we commit **after durable progress**.

## Storage
- Postgres 15+
  - control-plane: `clients`, `rules` (rule-service)
  - data-plane: `notifications` (aggregator/sender)
- Redis 7+
  - `rules:snapshot` (serialized rule indexes)
  - `rules:version` (monotonic integer)

## Tooling
- Migrations: `golang-migrate/migrate`
- Lint: `golangci-lint`
- Build/run: Makefile per service + root docker-compose

## Infrastructure Management
- **Centralized**: All infrastructure (Postgres, Kafka, Redis, Zookeeper) is managed centrally
- **Root docker-compose.yml**: Defines shared infrastructure with consistent container names
- **Verification scripts**: Services verify dependencies exist, but don't manage them
- **Migration runner**: Centralized script runs all migrations from all services
- **Topic creation**: Centralized script creates all Kafka topics
- See `docs/architecture/INFRASTRUCTURE.md` for full details

## Migration Strategy
- **Shared database**: All services use same Postgres database (`alerting`)
- **Coordinated versioning**: Services use sequential migration numbers to avoid conflicts
  - rule-service: 000001-000005 (control-plane tables)
  - aggregator: 000006+ (data-plane tables)
- **Validation**: Run `make check-migrations` from root to validate consistency
- **Documentation**: See `migrations/MIGRATION_STRATEGY.md` for full strategy

## Event format
- Protobuf messages defined in `proto/*.proto`, generated Go types in `pkg/proto`.
- Stable IDs:
  - `alert_id`: provided by producer; else hash of `(severity,source,name,event_ts)`
  - `rule_id`: UUID from DB
  - `notification_id`: DB-generated
