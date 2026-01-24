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

## Production Deployment
- **Platform**: AWS ECS with EC2 launch type
- **Infrastructure as Code**: Terraform (see `terraform/`)
- **Container Registry**: Amazon ECR
- **Database**: RDS Postgres 15 (db.t3.micro free tier eligible)
- **Cache**: ElastiCache Redis 7 (cache.t3.micro free tier eligible)
- **Messaging**: Kafka on ECS (self-hosted, 6 partitions per topic)
- **API Gateway**: AWS HTTP API for HTTPS endpoints
- **Scaling**: Auto-scaling based on CPU/memory (1-3 instances per service)
- **CI/CD**: GitHub Actions for automated deployments
- **Monitoring**: CloudWatch Logs and Metrics
- See `docs/deployment/PRODUCTION_DEPLOYMENT.md` for full guide

### Current Configuration (Ultra-Low-Cost)
- **EC2**: 1x t3.small (2 vCPU, 2GB RAM) - ~$15/month
- **Container Memory**: 150 MB per service
- **Kafka Partitions**: 6 per topic (via `KAFKA_NUM_PARTITIONS`)
- **Task Counts**:
  - Evaluator: 2 instances
  - Aggregator: 2 instances
  - Sender: 1 instance
  - Others: 1 instance each

### Performance Limits (current config)
| Metric | Value | Notes |
|--------|-------|-------|
| Producer | ~780/s | Kafka broker limit |
| Evaluator | ~320/s | 2 instances |
| Aggregator | ~100/s | 2 instances |
| Sender | ~120/s | Rate limited |
| Total Pipeline | ~320/s | Evaluator is bottleneck |
