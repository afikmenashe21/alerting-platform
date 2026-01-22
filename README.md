# Alerting Platform

A multi-service Go application implementing an end-to-end alert notification platform with Kafka, Postgres, and Redis.

**Status**: ✅ Deployed to AWS ECS - Production-ready MVP

## Quick Start

**Production Deployment**: See [`docs/deployment/CURRENT_STATUS.md`](docs/deployment/CURRENT_STATUS.md) for AWS ECS deployment status and next steps.

**Local Development**:

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

## Production Deployment (AWS)

Deploy to AWS ECS with Terraform:

```bash
cd terraform
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars - set db_password
terraform init && terraform apply

# Build and push Docker images
./scripts/deployment/build-and-push.sh

# Update ECS services
./scripts/deployment/update-services.sh
```

See `docs/deployment/` for complete guides:
- `QUICKSTART.md` - Deploy in 30 minutes
- `PRODUCTION_DEPLOYMENT.md` - Full production guide
- `PREREQUISITES.md` - Required tools and credentials
- `ULTRA_LOW_COST.md` - Deploy for ~$5-10/month

## Documentation

Documentation is organized in the `docs/` directory:

- **Deployment** (`docs/deployment/`):
  - `QUICKSTART.md` - 30-minute deployment guide
  - `PRODUCTION_DEPLOYMENT.md` - Complete production guide
  - `PREREQUISITES.md` - Tools and credentials setup
  - `ULTRA_LOW_COST.md` - Ultra-low-cost configuration

- **Guides** (`docs/guides/`):
  - `SETUP.md` - Complete local setup guide
  - `QUICKSTART.md` - Quick start instructions

- **Architecture** (`docs/architecture/`):
  - `INFRASTRUCTURE.md` - Infrastructure management details
  - `PROTOBUF_INTEGRATION_STRATEGY.md` - Protobuf design

- **Features** (`docs/features/`):
  - `WILDCARD_RULES_DESIGN.md` - Wildcard rules design
  - `WILDCARD_RULES_USAGE.md` - Wildcard rules usage

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
├── services/           # All services (with Dockerfiles)
│   ├── rule-service/
│   ├── rule-updater/
│   ├── evaluator/
│   ├── aggregator/
│   ├── sender/
│   └── alert-producer/
├── terraform/         # AWS infrastructure (Terraform)
│   ├── main.tf
│   └── modules/       # VPC, ECS, RDS, Redis, Kafka, ALB
├── docs/              # Documentation
│   ├── deployment/    # Production deployment guides
│   ├── guides/        # Setup and quick start guides
│   ├── architecture/  # Architecture docs
│   └── features/      # Feature documentation
├── scripts/           # Centralized scripts
│   ├── infrastructure/  # Setup and verify scripts
│   ├── migrations/      # DB migration scripts
│   └── deployment/      # Build/push/update scripts
├── migrations/        # Database schema (init-schema.sql)
├── memory-bank/       # Project memory bank (design decisions)
├── proto/             # Protobuf definitions
├── pkg/               # Shared Go packages
├── rule-service-ui/   # React UI for rule-service
├── docker-compose.yml # Local infrastructure
└── Makefile           # Root-level commands
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
