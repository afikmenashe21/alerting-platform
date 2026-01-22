# Alerting Platform – Active Context

## What we’re doing right now
We are building the MVP services in Go and want a Cursor “Memory Bank” to capture the exact design decisions and contracts.

Completed:
- ✅ Event contracts + topic names defined (protobuf messages)
- ✅ alert-producer: generate alerts + load tests
- ✅ evaluator: warmup + matching + `alerts.matched` output (one message per client_id)
- ✅ aggregator: idempotent insert + `notifications.ready` output

Recently Completed:
1) ✅ Postgres migrations for clients/rules (rule-service)
2) ✅ rule-service: CRUD + publish `rule.changed` events
3) ✅ rule-updater: snapshot writer to Redis (consumes rule.changed, rebuilds snapshot, increments version)
4) ✅ sender: consume notifications.ready + send via email (SMTP), Slack (webhook API), and webhook (HTTP POST) + update status
5) ✅ rule-service-ui: React UI for managing clients, rules, and endpoints
6) ✅ Centralized infrastructure management (Postgres, Kafka, Redis, Zookeeper)
7) ✅ Protobuf Integration: All Kafka topics now use protobuf messages defined in `proto/*.proto` with generated Go types in `pkg/proto/`
8) ✅ Protobuf Enhanced Tooling: Added buf linting, breaking change detection, code verification, CI/CD integration, and pre-commit hooks
9) ✅ Protobuf Severity Alignment: Changed protobuf enum values to match database format (LOW, MEDIUM, HIGH, CRITICAL)
10) ✅ **Production AWS Deployment (2026-01-21)**:
    - Created Terraform infrastructure for AWS ECS (EC2 launch type)
    - Built and pushed Docker images for all 6 services to ECR
    - Deployed: VPC, ECS cluster, RDS Postgres, ElastiCache Redis, self-hosted Kafka
    - All 8 ECS services running (kafka, zookeeper, + 6 application services)
    - Ultra-low-cost configuration (~$15-20/month, no ALB due to AWS account restriction)
    - Increased Kafka partitions to 9 for better parallelism
    - Comprehensive deployment documentation created

Current State (2026-01-22 20:50 UTC):
- Infrastructure: ✅ Fully deployed to AWS (VPC, ECS, RDS, ElastiCache)
- Docker Images: ✅ Built and pushed to ECR (linux/amd64, latest tag)
- ECS Container Instances: ✅ 2x t3.small instances registered (public subnets)
- Database Schema: ✅ All tables created via ECS migration task
- ECS Services: ✅ **ALL STABLE AND RUNNING** (2026-01-22 20:50 UTC)
  - Kafka: ✅ RUNNING - Connected to Zookeeper
  - Zookeeper: ✅ RUNNING - Listening on port 2181
  - rule-service: ✅ RUNNING - 1/1 tasks, deployment COMPLETED
  - evaluator: ✅ RUNNING - 1/1 tasks, deployment COMPLETED
  - aggregator: ✅ RUNNING - 1/1 tasks, deployment COMPLETED
  - sender: ✅ RUNNING - 1/1 tasks, deployment COMPLETED
  - rule-updater: ✅ RUNNING - 1/1 tasks, deployment COMPLETED
  - alert-producer: Scaled to 0 (intentional - test generator)
- Load Balancer: ❌ NOT DEPLOYED - ALB disabled to reduce costs
- Kafka Topics: ✅ **COMPLETED** (2026-01-22 19:13 UTC) - All 4 topics have 9 partitions
  - alerts.new: 9 partitions ✅
  - rule.changed: 9 partitions ✅
  - alerts.matched: 9 partitions ✅
  - notifications.ready: 9 partitions ✅

## Recent Infrastructure Fixes (2026-01-21)
1. **AMI Fix**: Changed from hardcoded AMI to SSM parameter for ECS-optimized AMI
2. **Network Mode**: Changed from `awsvpc` to `bridge` mode (avoids ENI limits on t3.micro)
3. **Subnet Fix**: ECS instances moved to public subnets (no NAT Gateway needed = saves $32/month)
4. **Memory Reduction**: Kafka/Zookeeper memory reduced to fit on t3.micro (384MB/512MB)
5. **Build Platform**: Docker images now built with `--platform linux/amd64`

## Database Migration Completed (2026-01-21)

Successfully ran database schema migration using Docker-based ECS task:
- Created migration Docker image with PostgreSQL client
- Pushed to ECR: `248508119478.dkr.ecr.us-east-1.amazonaws.com/alerting-platform-prod-migration`
- Ran as one-off ECS task with environment variables for DB connection
- All 4 tables created successfully: clients, endpoints, notifications, rules
- Migration artifacts: `migrations/Dockerfile`, `migrations/docker-entrypoint.sh`
- Script: `scripts/deployment/run-migration.sh`

## ✅ RESOLVED: Production Deployment Issues (2026-01-22)

### Issue 1: Kafka-Zookeeper Connection (FIXED 18:00 UTC)
**Problem**: `java.net.ConnectException: Connection refused` to Zookeeper
**Cause**: Startup timing with separate ECS tasks using host networking
**Fix**: Restarted services in correct order with delays
**Result**: ✅ All services connected and stable

### Issue 2: Duplicate Tasks and Deployment Loop (FIXED 20:50 UTC)
**Problem**: rule-service stuck with 2 tasks running, deployment IN_PROGRESS
**Cause**: Docker health checks failing (no `wget` in Alpine images), causing ECS to continuously start replacement tasks
**Fix**: 
1. Updated `terraform/modules/ecs-service/main.tf` to remove health checks
2. Created new task definition revision 7 without health check
3. Updated service to use new task definition
4. Stopped old tasks to break the loop

**Result**: ✅ All services now 1/1 tasks, all deployments COMPLETED

### Key Decisions Made
1. **No ALB**: Disabled to save ~$16/month (no load balancer needed for internal services)
2. **No Health Checks**: Removed ECS Docker health checks (not needed without ALB)
3. **rule-service Architecture**: Confirmed single service does both HTTP API + Kafka publishing

### Files Created/Updated
- ✅ `DEPLOYMENT_ANALYSIS.md` - Root cause analysis and solutions
- ✅ `DEPLOYMENT_FIX_SUMMARY.md` - Complete fix summary
- ✅ `QUICK_STATUS.md` - Quick reference guide
- ✅ `scripts/deployment/fix-kafka-zookeeper.sh` - Service restart script
- ✅ `scripts/deployment/kafka-topics-commands.sh` - Topic creation guide
- ✅ `terraform/modules/ecs-service/main.tf` - Removed health checks

### Next Steps
1. ~~Create Kafka Topics~~ ✅ **COMPLETED** (2026-01-22 19:13 UTC)

2. **Test End-to-End Flow** (READY):
   - All infrastructure is ready
   - All services are running
   - Kafka topics configured with 9 partitions
   - Create test client and rule
   - Generate test alert
   - Verify notifications

3. **Long-term Improvements**:
   - Combine Kafka+Zookeeper into single task (prevent startup issues)
   - Add `wget` or `curl` to Dockerfiles for future health checks
   - Enable ALB when budget allows

## AWS Resources Summary
- **ECS Cluster**: `alerting-platform-prod-cluster`
- **EC2 Instances**: 2-3x t3.micro in public subnets
- **RDS Endpoint**: `alerting-platform-prod-postgres.cot8kqgoccg6.us-east-1.rds.amazonaws.com:5432`
- **Redis Endpoint**: `alerting-platform-prod-redis.ves3x9.0001.use1.cache.amazonaws.com:6379`
- **Kafka**: `kafka.alerting-platform-prod.local:9092` (internal via service discovery)
- **Cost**: ~$15-20/month (no NAT Gateway, no ALB)

## Future Tasks
1) 🔄 UI Integration for alert-producer (HTTP API wrapper + UI component)

## Code health
- Completed comprehensive cleanup and modularization across all services:
  - Extracted redundant code patterns (validation, error handling, database scanning)
  - Split large files (>200 lines) by resource/concern where appropriate
  - All services maintain existing functionality with improved organization
  - Remaining files slightly over 200 lines are well-organized handler files without obvious redundancy

## Decisions locked for MVP
- Rules support exact match and wildcard "*" on (severity, source, name).
- Dedupe in aggregator DB unique constraint.
- Redis snapshot warm start (no evaluator DB reads).
- Protobuf messages for all Kafka topics (no JSON wire format on Kafka).
- Evaluator output: one message per client_id (keyed by client_id for tenant locality).
- Wildcard support: "*" matches any value for that field, enabling multiple rules per client to match same alert.
