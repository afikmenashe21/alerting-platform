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

### ✅ Production Testing Completed (2026-01-22 19:35 UTC)

**Test Results**:
- ✅ rule-service API accessible and functional
- ✅ Client creation working
- ✅ Rule creation working  
- ✅ Endpoint creation working
- ✅ All 7 core services running and stable

**Issues Fixed During Testing**:
1. ✅ Security group missing ephemeral port range (32768-65535) - Added for ECS dynamic port mappings
2. ✅ Task definition port mismatch - Updated rule-service from containerPort 8080 to 8081
3. ✅ Terraform port configuration - Updated main.tf to reflect correct service ports

**Test Script**: `scripts/test/test-production-simple.sh`

### ✅ Kafka Connectivity Fixed (2026-01-22 20:10 UTC)

**Root Cause**: Hardcoded IP addresses in Kafka configuration - when Kafka moved instances (10.0.1.109 → 10.0.0.117 → 10.0.101.19), all consumer services failed to connect.

**Solution Implemented**: AWS Cloud Map DNS-based service discovery
- Service Discovery Namespace: `alerting-platform-prod.local` (ID: ns-jhsxaal5lpovw57m)
- Kafka DNS Name: `kafka.alerting-platform-prod.local:9092` (A record, auto-updated)
- Combined Kafka + Zookeeper into single task (ensures co-location, host network mode)
- Updated Kafka advertised listeners: DNS name (not IP)
- Updated all consumer services: DNS-based broker address

**Results**:
- ✅ All 4 consumer groups connected: evaluator-group, rule-updater-group, aggregator-group, sender-group
- ✅ Zero Kafka connection errors across all services
- ✅ DNS resolves to current IP automatically
- ✅ No manual intervention needed for future instance changes

**Architecture Changes**:
- Deprecated: Standalone kafka and zookeeper ECS services (scaled to 0)
- New: kafka-combined service (awsvpc mode, service discovery enabled)
- Network: Kafka uses awsvpc mode for proper A record support
- Services: evaluator (rev 9), rule-updater (rev 8), aggregator (rev 8), sender (rev 8), rule-service (rev 9)

**Documentation**: `docs/deployment/KAFKA_FIX_SUMMARY.md`

### Next Steps
1. ~~Create Kafka Topics~~ ✅ **COMPLETED** (2026-01-22 19:13 UTC)
2. ~~Test End-to-End Flow~~ ✅ **COMPLETED** (2026-01-22 19:35 UTC)

3. **Full E2E Alert Processing Test** (READY):
   - Scale up alert-producer
   - Generate test alerts
   - Verify alerts flow through: evaluator → aggregator → sender
   - Check notifications in database

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

## UI Deployment Infrastructure (2026-01-23)

Deployed free-tier infrastructure for rule-service-ui:

### Architecture (Direct EC2 Access + GitHub Pages)
- **GitHub Pages**: Static site hosting (free)
- **Direct EC2 Access**: Services exposed on public Elastic IP (no Lambda/API Gateway needed)
- **Elastic IP**: `34.201.202.8` (stable, never changes)

### API Endpoints
- **rule-service**: `http://34.201.202.8:8081`
- **alert-producer**: `http://34.201.202.8:8082`

### Changes Made
- Switched rule-service and alert-producer to **host network mode** (fixed ports)
- Opened security group for ports 8081 and 8082 from anywhere
- Added Elastic IP with auto-association on EC2 instance startup
- UI uses `VITE_RULE_SERVICE_URL` and `VITE_ALERT_PRODUCER_URL` environment variables

### Key Files
- `.github/workflows/deploy-ui.yml` - GitHub Actions workflow (input: EC2 IP)
- `rule-service-ui/src/services/api.js` - Configurable API endpoints
- `terraform/modules/ecs-cluster/main.tf` - Elastic IP and security group

### Deployment Complete
- UI: `https://afikmenashe21.github.io/alerting-platform/`
- Backend APIs: Direct access via Elastic IP

### Cost Estimate
- GitHub Pages: Free
- Elastic IP: Free (when attached to running instance)
- **Total: $0/month additional**

## Next Steps
1) ✅ UI deployed to GitHub Pages
2) ✅ Backend accessible via Elastic IP
3) ✅ Sender service migrated from SMTP to AWS SES API (2026-01-23)

## Code health
- Completed comprehensive cleanup and modularization across all services:
  - Extracted redundant code patterns (validation, error handling, database scanning)
  - Split large files (>200 lines) by resource/concern where appropriate
  - All services maintain existing functionality with improved organization
  - Remaining files slightly over 200 lines are well-organized handler files without obvious redundancy

## Centralized Metrics System (2026-01-24)

### Overview
Centralized metrics collection and reporting via `pkg/metrics/` package. All 6 services now properly integrate with the shared metrics system.

### Centralized Package (`pkg/metrics/metrics.go`)
- **Collector**: Writes metrics to Redis every 30s with `metrics:` key prefix
- **Reader**: Reads service metrics from Redis (used by rule-service UI)
- **Helper Functions** (newly added to reduce duplication):
  - `GetEnvOrDefault(key, default)` - environment variable lookup
  - `MaskDSN(dsn)` - mask sensitive DSN info for logging
  - `ConnectRedis(ctx, addr)` - standard Redis connection with ping validation

### Metrics Tracked Per Service
- **Standard Counters**: MessagesReceived, MessagesProcessed, MessagesPublished, ProcessingErrors
- **Rates**: MessagesPerSecond (calculated per report interval)
- **Latencies**: AvgProcessingLatencyMs
- **Custom Counters**: Service-specific metrics (e.g., `alerts_matched`, `notifications_sent`)

### Service Integration Status
| Service | Metrics Collector | Custom Counters |
|---------|-------------------|-----------------|
| evaluator | ✅ Full integration | alerts_matched, alerts_unmatched, rules_count |
| aggregator | ✅ Full integration | notifications_created, notifications_deduplicated |
| sender | ✅ Full integration | notifications_sent, notifications_failed, notifications_skipped |
| rule-updater | ✅ Full integration | rules_CREATED, rules_UPDATED, rules_DELETED, rules_DISABLED |
| alert-producer | ✅ Full integration | (uses standard published/errors) |
| rule-service | ✅ Reader + Collector | (serves as metrics API endpoint) |

### Code Deduplication
Removed duplicated helper functions from all 6 services:
- `getEnvOrDefault()` → now use `metrics.GetEnvOrDefault()`
- `maskDSN()` → now use `metrics.MaskDSN()`
- Redis connection boilerplate → now use `metrics.ConnectRedis()`

### API Endpoints
- `GET /api/v1/metrics` - Database metrics (notifications, rules, clients, endpoints)
- `GET /api/v1/services/metrics` - All service metrics from Redis
- `GET /api/v1/services/metrics?service=<name>` - Single service metrics

## Decisions locked for MVP
- Rules support exact match and wildcard "*" on (severity, source, name).
- Dedupe in aggregator DB unique constraint.
- Redis snapshot warm start (no evaluator DB reads).
- Protobuf messages for all Kafka topics (no JSON wire format on Kafka).
- Evaluator output: one message per client_id (keyed by client_id for tenant locality).
- Wildcard support: "*" matches any value for that field, enabling multiple rules per client to match same alert.
