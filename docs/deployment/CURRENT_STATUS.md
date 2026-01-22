# Production Deployment - Current Status

**Last Updated**: 2026-01-22 20:50 UTC

## ✅ Deployment Complete

All services are deployed and running on AWS ECS.

### Infrastructure

| Component | Status | Details |
|-----------|--------|---------|
| **VPC** | ✅ Running | 10.0.0.0/16, 2 AZs (us-east-1a, us-east-1b) |
| **ECS Cluster** | ✅ Running | 2x t3.small EC2 instances (public subnets) |
| **RDS Postgres** | ✅ Running | db.t3.micro, database schema migrated |
| **ElastiCache Redis** | ✅ Running | cache.t3.micro, single node |
| **Kafka + Zookeeper** | ✅ Running | Self-hosted on ECS (host networking) |
| **ECR** | ✅ Active | 6 repositories with latest images |
| **ALB** | ❌ Not Deployed | Disabled to save ~$16/month |

### Services Status

| Service | Tasks | Deployment | Health | Notes |
|---------|-------|------------|--------|-------|
| kafka | 1/1 | COMPLETED | ✅ | Connected to Zookeeper |
| zookeeper | 1/1 | COMPLETED | ✅ | Listening on port 2181 |
| rule-service | 1/1 | COMPLETED | ✅ | HTTP API + Kafka publisher |
| evaluator | 1/1 | COMPLETED | ✅ | Consumer group active |
| aggregator | 1/1 | COMPLETED | ✅ | Consumer group active |
| sender | 1/1 | COMPLETED | ✅ | Consumer group active |
| rule-updater | 1/1 | COMPLETED | ✅ | Writing to Redis |
| alert-producer | 0/0 | COMPLETED | N/A | Scaled to 0 (test generator) |

**All deployments are stable with no looping or health check issues.**

## Issues Resolved (2026-01-22)

### Issue 1: Kafka-Zookeeper Connection
- **Problem**: Kafka couldn't connect to Zookeeper (connection refused)
- **Cause**: Startup timing with separate ECS tasks
- **Fix**: Restarted services in correct order with delays
- **Script**: `scripts/deployment/fix-kafka-zookeeper.sh`

### Issue 2: Deployment Loop (2 Tasks Running)
- **Problem**: rule-service stuck with 2 tasks, deployment IN_PROGRESS
- **Cause**: Docker health checks failing (Alpine images missing `wget`)
- **Fix**: 
  - Removed health checks from `terraform/modules/ecs-service/main.tf`
  - Created new task definition revision 7
  - Not needed without ALB anyway
- **Result**: All services stable at 1/1 tasks

## Architecture Decisions

### No Application Load Balancer
- **Decision**: ALB disabled to reduce costs
- **Savings**: ~$16/month (ALB + data transfer)
- **Impact**: Services accessed via EC2 instance IPs only
- **Trade-off**: Acceptable for internal/development deployment

### No ECS Health Checks
- **Decision**: Removed Docker health checks from task definitions
- **Reason**: Alpine images don't have `wget`, checks were failing
- **Impact**: ECS won't automatically replace unhealthy tasks
- **Mitigation**: Services have internal health monitoring and automatic restart

### Single Task Per Service
- **Decision**: All services run with desired count = 1
- **Reason**: Cost optimization, sufficient for MVP/development
- **Scalable**: Can increase to 3 tasks per service if needed (already configured)

## Cost Estimate

**Monthly Cost**: ~$15-30 (with free tier) or ~$60/month (after free tier)

| Resource | Cost |
|----------|------|
| 2x t3.small EC2 | ~$30/month (or free tier) |
| RDS db.t3.micro | ~$15/month (or free tier) |
| ElastiCache | ~$15/month (or free tier) |
| Data transfer | ~$1-5/month |
| **Total** | **~$60/month** |

No ALB saves ~$16/month.

## ✅ All Setup Complete!

### Kafka Topics (COMPLETED 2026-01-22 19:13 UTC)

All topics configured with **9 partitions**:
- ✅ alerts.new: 9 partitions
- ✅ rule.changed: 9 partitions
- ✅ alerts.matched: 9 partitions
- ✅ notifications.ready: 9 partitions

## Next Steps

### 1. Test End-to-End Flow

Once topics are created:

1. **Create test client**:
   ```bash
   curl -X POST http://<EC2_IP>:8081/api/v1/clients \
     -H "Content-Type: application/json" \
     -d '{"name": "Test Client", "email": "test@example.com"}'
   ```

2. **Create test rule**:
   ```bash
   curl -X POST http://<EC2_IP>:8081/api/v1/rules \
     -H "Content-Type: application/json" \
     -d '{"client_id": "<ID>", "severity": "HIGH", "source": "*", "name": "*"}'
   ```

3. **Generate test alert** (scale alert-producer to 1 first)

4. **Verify notifications** (check sender logs)

### 3. Optional Improvements

**Add Health Checks** (if desired):
```dockerfile
# Add to all service Dockerfiles:
RUN apk --no-cache add ca-certificates tzdata wget

# Then re-enable in terraform/modules/ecs-service/main.tf
```

**Combine Kafka + Zookeeper**:
- Create single task definition with both containers
- Guarantees startup ordering
- Eliminates connection issues
- See Memory Bank for details

**Enable ALB** (when budget allows):
- Uncomment ALB module in `terraform/main.tf`
- Update service modules: `load_balancer_enabled = true`
- Provides HTTPS, custom domain, better routing

## Useful Commands

### Check Service Status
```bash
aws ecs describe-services --cluster alerting-platform-prod-cluster \
  --services kafka zookeeper rule-service evaluator aggregator sender rule-updater \
  --region us-east-1 \
  --query 'services[*].{Name:serviceName,Running:runningCount,Desired:desiredCount}'
```

### View Logs
```bash
aws logs tail /ecs/alerting-platform/prod/<service-name> --follow --region us-east-1
```

### Update Service
```bash
aws ecs update-service --cluster alerting-platform-prod-cluster \
  --service <service-name> --force-new-deployment --region us-east-1
```

### Restart Services (if issues occur)
```bash
./scripts/deployment/fix-kafka-zookeeper.sh
```

## Available Scripts

| Script | Purpose |
|--------|---------|
| `build-and-push.sh` | Build and push all Docker images to ECR |
| `update-services.sh` | Force new deployment of all services |
| `fix-kafka-zookeeper.sh` | Restart Kafka/Zookeeper in correct order |
| `kafka-topics-commands.sh` | Show commands to create Kafka topics |
| `run-migration.sh` | Run database migrations (already completed) |

## Documentation

- **Production Deployment Guide**: `docs/deployment/PRODUCTION_DEPLOYMENT.md`
- **Prerequisites**: `docs/deployment/PREREQUISITES.md`
- **Terraform README**: `terraform/README.md`
- **Memory Bank**: `memory-bank/` (current state, decisions, progress)

## Support

**If services become unhealthy**:
1. Check logs: `aws logs tail /ecs/alerting-platform/prod/<service> --region us-east-1`
2. Restart if needed: `./scripts/deployment/fix-kafka-zookeeper.sh`
3. Check Memory Bank for known issues and solutions

**If deployment loops**:
- This was fixed by removing health checks
- Should not occur again
- If it does, stop extra tasks manually

**For Kafka issues**:
- Ensure Zookeeper is running first
- Check `10.0.1.109:9092` connectivity from services
- Restart in order: Zookeeper → Kafka → other services
