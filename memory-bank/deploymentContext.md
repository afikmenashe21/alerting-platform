# Alerting Platform – Deployment Context

## Production Deployment Architecture

### Platform: AWS ECS (Elastic Container Service)

We chose **ECS with EC2 launch type** over Kubernetes because:
- Simpler to manage for our scale
- Better integration with AWS services
- Free tier eligible (t3.micro instances)
- Easier learning curve
- Good balance between Docker Compose and full Kubernetes

### Infrastructure Overview

```
┌─────────────────────────────────────────────────┐
│                   AWS Cloud                     │
│                                                 │
│  ┌──────────────────────────────────────────┐  │
│  │  VPC (10.0.0.0/16)                       │  │
│  │                                           │  │
│  │  ┌─────────────┐     ┌─────────────┐    │  │
│  │  │   Public    │     │   Public    │    │  │
│  │  │  Subnet     │     │  Subnet     │    │  │
│  │  │  (AZ-a)     │     │  (AZ-b)     │    │  │
│  │  │             │     │             │    │  │
│  │  │  ┌──────────────────────────┐   │    │  │
│  │  │  │   Load Balancer (ALB)    │   │    │  │
│  │  │  └──────────────────────────┘   │    │  │
│  │  └─────────────┘     └─────────────┘    │  │
│  │         │                   │            │  │
│  │  ┌─────────────┐     ┌─────────────┐    │  │
│  │  │  Private    │     │  Private    │    │  │
│  │  │  Subnet     │     │  Subnet     │    │  │
│  │  │  (AZ-a)     │     │  (AZ-b)     │    │  │
│  │  │             │     │             │    │  │
│  │  │  ECS Tasks: │     │  ECS Tasks: │    │  │
│  │  │  - Services │     │  - Services │    │  │
│  │  │  - Kafka    │     │  - Kafka    │    │  │
│  │  │  - Redis    │     │  - Redis    │    │  │
│  │  │             │     │             │    │  │
│  │  │  RDS (Multi-AZ)   │  ElastiCache│    │  │
│  │  └─────────────┘     └─────────────┘    │  │
│  └──────────────────────────────────────────┘  │
└─────────────────────────────────────────────────┘
```

### Components

1. **VPC and Networking**
   - VPC: 10.0.0.0/16
   - 2 Public Subnets (for ALB)
   - 2 Private Subnets (for ECS tasks, RDS, Redis)
   - NAT Gateway for outbound internet from private subnets
   - Internet Gateway for public subnets

2. **ECS Cluster**
   - EC2 launch type (not Fargate - better control and costs)
   - t3.micro instances (free tier eligible)
   - Auto Scaling Group: 1-4 instances
   - Amazon Linux 2 ECS-optimized AMI

3. **Services on ECS**
   - rule-service (port 8080, behind ALB)
   - alert-producer (port 8081, behind ALB)
   - evaluator (internal only)
   - aggregator (internal only)
   - sender (internal only)
   - rule-updater (internal only, ALWAYS 1 instance)
   - Kafka + Zookeeper (self-hosted on ECS)

4. **Managed Services**
   - **RDS Postgres 15**: db.t3.micro, 20GB, Multi-AZ optional
   - **ElastiCache Redis 7**: cache.t3.micro, single node
   - **ECR**: 6 repositories (one per service)

5. **Load Balancing**
   - Application Load Balancer (ALB)
   - Target groups for rule-service (8080) and alert-producer (8081)
   - Health checks on `/health` endpoints

6. **Auto Scaling**
   - ECS Service Auto Scaling (1-3 tasks per service)
   - EC2 Auto Scaling (1-4 instances in cluster)
   - Target tracking: 70% CPU, 80% Memory

## Scaling Capabilities

### Service Scaling

| Service | Min | Default | Max | Notes |
|---------|-----|---------|-----|-------|
| rule-service | 1 | 1 | 3 | Stateless, safe to scale |
| evaluator | 1 | 1 | 3 | Stateless, loads from Redis |
| aggregator | 1 | 1 | 3 | Idempotent via DB constraint |
| sender | 1 | 1 | 3 | Idempotent sends |
| alert-producer | 1 | 1 | 3 | Test/load generator |
| rule-updater | 1 | 1 | **1** | **NEVER SCALE** - writes Redis snapshot |

### Kafka Partitions

- **9 partitions** per topic (increased from 3)
- Allows up to 9 consumer instances per service
- Topics: `alerts.new`, `rule.changed`, `alerts.matched`, `notifications.ready`

### Auto-Scaling Triggers

Services auto-scale based on:
- **CPU > 70%**: Scale out
- **Memory > 80%**: Scale out
- Cool-down: 60s scale-out, 300s scale-in

## Deployment Workflow

### CI/CD Pipeline (GitHub Actions)

```
┌─────────────┐
│  Git Push   │
│  to main    │
└──────┬──────┘
       │
       ▼
┌─────────────────────┐
│  GitHub Actions     │
│  - Build images     │
│  - Push to ECR      │
│  - Update ECS       │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│  ECS Deployment     │
│  - Rolling update   │
│  - Health checks    │
│  - Rollback on fail │
└─────────────────────┘
```

### Manual Deployment

```bash
# 1. Build and push images
./scripts/deployment/build-and-push.sh

# 2. Update ECS services
./scripts/deployment/update-services.sh

# 3. Monitor deployment
aws ecs describe-services --cluster alerting-platform-prod-cluster \
  --services rule-service evaluator aggregator sender rule-updater alert-producer
```

## Infrastructure Management

### Terraform Modules

```
terraform/
├── main.tf                    # Root module
├── variables.tf               # Input variables
├── outputs.tf                 # Outputs (ALB DNS, ECR URLs, etc.)
├── terraform.tfvars.example   # Example configuration
└── modules/
    ├── vpc/                   # Network infrastructure
    ├── ecr/                   # Container registry
    ├── ecs-cluster/           # ECS cluster and EC2 instances
    ├── rds/                   # Postgres database
    ├── redis/                 # ElastiCache Redis
    ├── kafka/                 # Kafka on ECS
    ├── alb/                   # Load balancer
    └── ecs-service/           # Reusable service module
```

### Deployment Commands

```bash
# Initialize
cd terraform
terraform init

# Plan
terraform plan

# Apply
terraform apply

# Destroy
terraform destroy
```

## Cost Optimization

### Free Tier Usage (First 12 Months)

- ✅ 2x t3.micro EC2 (750 hours/month)
- ✅ RDS db.t3.micro (750 hours/month, 20GB)
- ✅ ElastiCache cache.t3.micro (if available)
- ✅ 750 hours ALB
- ❌ NAT Gateway (~$32/month - NOT free)

### Cost Reduction Strategies

1. **Single EC2 Instance**: `ecs_desired_capacity = 1`
2. **Remove NAT Gateway**: Set `create_nat_gateway = false` in VPC module
3. **Single Service Instance**: `service_desired_count = 1`
4. **Smaller Containers**: `container_cpu = 128`, `container_memory = 256`
5. **Short Log Retention**: `log_retention_days = 1`

**Estimated monthly cost**: $35-50 (with free tier), $100-110 (after free tier)

## Security

### Network Security

- Private subnets for ECS tasks, RDS, Redis
- Security groups with least privilege
- No public IPs on ECS tasks
- ALB in public subnets only

### Data Security

- RDS encryption at rest (enabled)
- RDS automated backups (7 days)
- Secrets in environment variables (should move to AWS Secrets Manager)
- HTTPS on ALB (requires ACM certificate - not configured yet)

### IAM Roles

- ECS Task Execution Role: Pull images from ECR, write logs
- ECS Task Role: Application permissions
- EC2 Instance Role: Join ECS cluster

## Monitoring and Logging

### CloudWatch Logs

Each service logs to:
```
/ecs/alerting-platform/prod/<service-name>
```

Retention: 7 days (configurable)

### CloudWatch Metrics

- ECS Service metrics (CPU, Memory, Task count)
- ALB metrics (Request count, Target response time)
- RDS metrics (CPU, Connections, Storage)
- ElastiCache metrics (CPU, Memory, Connections)

### Container Insights

Disabled by default (costs money). Enable with:
```hcl
enable_container_insights = true
```

## Limitations and Constraints

### Current Limitations

1. **Single Kafka broker**: No replication, single point of failure
2. **Single Redis node**: No failover, single point of failure
3. **HTTP only**: No HTTPS configured (need ACM certificate)
4. **No custom domain**: Using ALB DNS name
5. **No WAF**: No web application firewall
6. **Manual migrations**: Database migrations run manually
7. **No blue/green deployments**: Using rolling updates only

### Production Readiness Checklist

Before production use:
- [ ] Enable HTTPS (ACM certificate + Route 53)
- [ ] Set strong database password (use AWS Secrets Manager)
- [ ] Enable RDS Multi-AZ
- [ ] Set up CloudWatch alarms
- [ ] Configure log aggregation
- [ ] Enable VPC Flow Logs
- [ ] Set up AWS Backup
- [ ] Document runbooks
- [ ] Load test the platform
- [ ] Set up on-call rotation

## Critical Operational Notes

### rule-updater MUST NEVER SCALE

⚠️ **CRITICAL**: The `rule-updater` service writes Redis snapshots atomically. Running multiple instances would cause race conditions and corrupt the snapshot.

**Enforcement**:
- Terraform sets `desired_count = 1`, `max_count = 1`
- Auto-scaling is disabled for this service
- Manual scaling MUST NOT be performed

### Database Migrations

Migrations must be run manually before deploying new services:
1. Connect to RDS (via bastion or EC2 instance in VPC)
2. Run: `./scripts/migrations/run-migrations.sh`
3. Verify: Check migration version in DB

### Kafka Topic Management

Topics must be created before services start:
1. Connect to Kafka container in ECS
2. Run topic creation commands (9 partitions each)
3. Verify: `kafka-topics --list`

## Disaster Recovery

### Backup Strategy

- **RDS**: Automated daily backups (7 day retention)
- **Redis**: Daily snapshots (5 day retention)
- **Kafka**: No persistence (messages expire, acceptable for MVP)

### Recovery Procedures

1. **Database failure**: Restore from RDS automated backup
2. **Redis failure**: Rebuild snapshot from Postgres (rule-updater does this on startup)
3. **Kafka failure**: Restart Kafka task, services will reconnect
4. **Complete failure**: Run `terraform apply`, restore DB, recreate topics

### RTO/RPO

- **RTO** (Recovery Time Objective): ~30 minutes
- **RPO** (Recovery Point Objective): ~5 minutes (last RDS backup)

## Documentation References

- **Production Deployment Guide**: `docs/deployment/PRODUCTION_DEPLOYMENT.md`
- **Terraform README**: `terraform/README.md`
- **Architecture Diagrams**: `docs/architecture/INFRASTRUCTURE.md`
- **Cost Estimates**: `terraform/README.md` (Cost Estimates section)
