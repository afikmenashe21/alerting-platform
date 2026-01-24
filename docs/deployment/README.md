# Deployment Documentation

Complete guides for deploying the alerting platform to production on AWS.

**Current Status**: ✅ **DEPLOYED** to AWS ECS (2026-01-22)  
See [`CURRENT_STATUS.md`](CURRENT_STATUS.md) for deployment status and next steps.

## Quick Links

- **[Quick Start Guide](QUICKSTART.md)** - Deploy in under 30 minutes
- **[Complete Production Guide](PRODUCTION_DEPLOYMENT.md)** - Comprehensive deployment documentation
- **[Performance Scaling Guide](PERFORMANCE_SCALING.md)** - Load test results and scaling strategies
- **[Terraform README](../../terraform/README.md)** - Infrastructure as Code details

## Overview

The alerting platform deploys to AWS using:

- **ECS (Elastic Container Service)** - Container orchestration
- **RDS Postgres** - Managed database
- **ElastiCache Redis** - Managed cache
- **Kafka on ECS** - Self-hosted messaging
- **Terraform** - Infrastructure as Code
- **GitHub Actions** - CI/CD pipeline

## Architecture

```
┌──────────────────────────────────────────┐
│  Application Load Balancer (ALB)        │
│  - rule-service (port 8080)              │
│  - alert-producer (port 8081)            │
└──────────┬───────────────────────────────┘
           │
           ▼
┌──────────────────────────────────────────┐
│  ECS Cluster (2x t3.micro EC2)           │
│  ┌────────────────────────────────────┐  │
│  │  Services (1-3 instances each):    │  │
│  │  - rule-service                    │  │
│  │  - evaluator                       │  │
│  │  - aggregator                      │  │
│  │  - sender                          │  │
│  │  - alert-producer                  │  │
│  │  - rule-updater (ALWAYS 1)         │  │
│  │                                    │  │
│  │  Infrastructure:                   │  │
│  │  - Kafka + Zookeeper               │  │
│  └────────────────────────────────────┘  │
└──────────────────────────────────────────┘
           │
    ┌──────┴──────┐
    ▼             ▼
┌─────────┐   ┌─────────┐
│   RDS   │   │  Redis  │
│ Postgres│   │  Cache  │
└─────────┘   └─────────┘
```

## Deployment Options

### 1. Quick Start (Recommended for First-Time)

Follow **[QUICKSTART.md](QUICKSTART.md)** for a rapid deployment:

- ⏱️ Time: ~30 minutes
- 📝 Step-by-step commands
- ✅ Minimal configuration
- 🎯 Get running fast

### 2. Production Deployment (Recommended for Production Use)

Follow **[PRODUCTION_DEPLOYMENT.md](PRODUCTION_DEPLOYMENT.md)** for complete setup:

- 📚 Comprehensive guide
- 🔒 Security best practices
- 📊 Monitoring and logging
- 🛠️ Troubleshooting
- 💰 Cost optimization

## Prerequisites

| Tool | Version | Required? |
|------|---------|-----------|
| AWS Account | - | ✅ Yes |
| AWS CLI | Latest | ✅ Yes |
| Terraform | >= 1.5.0 | ✅ Yes |
| Docker | Latest | ✅ Yes |
| Git | Any | ✅ Yes |
| Make | Any | ⚠️ Optional |

## Deployment Workflow

```mermaid
graph TD
    A[Configure Terraform] --> B[terraform apply]
    B --> C[Build Docker Images]
    C --> D[Push to ECR]
    D --> E[Run Migrations]
    E --> F[Create Kafka Topics]
    F --> G[Update ECS Services]
    G --> H[Verify Deployment]
```

## Cost Breakdown

### Free Tier (First 12 Months)
- ✅ EC2 (2x t3.micro): Free
- ✅ RDS (db.t3.micro): Free
- ✅ ElastiCache: Free (if eligible)
- ✅ ALB: Partially free
- ❌ NAT Gateway: ~$32/month

**Total**: $35-50/month

### After Free Tier
- EC2: ~$15/month
- RDS: ~$15/month
- ElastiCache: ~$12/month
- NAT Gateway: ~$32/month
- ALB: ~$18/month
- Data transfer: ~$5-10/month

**Total**: $100-110/month

## File Structure

```
alerting-platform/
├── terraform/                     # Infrastructure as Code
│   ├── main.tf                    # Root configuration
│   ├── variables.tf               # Input variables
│   ├── outputs.tf                 # Outputs
│   ├── terraform.tfvars.example   # Example config
│   └── modules/                   # Terraform modules
│       ├── vpc/
│       ├── ecr/
│       ├── ecs-cluster/
│       ├── rds/
│       ├── redis/
│       ├── kafka/
│       ├── alb/
│       └── ecs-service/
│
├── services/                      # Service source code
│   ├── rule-service/
│   │   └── Dockerfile
│   ├── evaluator/
│   │   └── Dockerfile
│   ├── aggregator/
│   │   └── Dockerfile
│   ├── sender/
│   │   └── Dockerfile
│   ├── rule-updater/
│   │   └── Dockerfile
│   └── alert-producer/
│       └── Dockerfile
│
├── scripts/
│   ├── deployment/
│   │   ├── build-and-push.sh     # Build and push images
│   │   └── update-services.sh    # Update ECS services
│   └── migrations/
│       └── run-migrations.sh      # Run DB migrations
│
├── .github/
│   └── workflows/
│       └── deploy.yml             # GitHub Actions CI/CD
│
└── docs/
    └── deployment/
        ├── README.md              # This file
        ├── QUICKSTART.md          # Quick start guide
        └── PRODUCTION_DEPLOYMENT.md  # Complete guide
```

## Key Scripts

### Build and Push Docker Images

```bash
./scripts/deployment/build-and-push.sh
```

Builds all 6 services and pushes to ECR.

### Update ECS Services

```bash
./scripts/deployment/update-services.sh
```

Forces new deployment of all services.

### Run Database Migrations

```bash
export POSTGRES_DSN="postgres://user:pass@host/db?sslmode=require"
./scripts/migrations/run-migrations.sh
```

## Scaling Configuration

### Service Scaling

All services (except rule-updater) can scale from 1 to 3 instances:

```hcl
# In terraform.tfvars
service_desired_count = 1  # Start with 1
service_max_count = 3      # Can scale to 3
```

**Exception**: `rule-updater` MUST always be 1 instance (writes Redis snapshot).

### Infrastructure Scaling

ECS cluster can scale from 1 to 4 EC2 instances:

```hcl
# In terraform.tfvars
ecs_desired_capacity = 2  # Start with 2 for HA
ecs_min_size = 1
ecs_max_size = 4
```

### Kafka Partitions

9 partitions per topic (allows up to 9 consumer instances per service):

```bash
# Topics are created with 9 partitions
alerts.new: 9 partitions
rule.changed: 9 partitions
alerts.matched: 9 partitions
notifications.ready: 9 partitions
```

## CI/CD Pipeline

GitHub Actions automatically deploys on push to `main`:

1. Build Docker images
2. Push to ECR
3. Update ECS services
4. Wait for stability
5. Verify deployment

Configure secrets in GitHub:
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- `AWS_REGION`

See `.github/workflows/deploy.yml` for details.

## Monitoring

### CloudWatch Logs

```bash
# View service logs
aws logs tail /ecs/alerting-platform/prod/<service-name> --follow

# Search for errors
aws logs filter-log-events \
  --log-group-name /ecs/alerting-platform/prod/evaluator \
  --filter-pattern "ERROR"
```

### Service Health

```bash
# Check all services
aws ecs describe-services \
  --cluster alerting-platform-prod-cluster \
  --services rule-service evaluator aggregator sender rule-updater alert-producer
```

## Common Tasks

### Scale a Service

```bash
aws ecs update-service \
  --cluster alerting-platform-prod-cluster \
  --service evaluator \
  --desired-count 3
```

### View Logs

```bash
aws logs tail /ecs/alerting-platform/prod/evaluator --follow
```

### Restart a Service

```bash
aws ecs update-service \
  --cluster alerting-platform-prod-cluster \
  --service evaluator \
  --force-new-deployment
```

### Check Target Health (ALB)

```bash
aws elbv2 describe-target-health \
  --target-group-arn <ARN>
```

## Troubleshooting

### Services Won't Start

1. Check CloudWatch logs
2. Verify security groups
3. Check database/Kafka connectivity
4. Verify environment variables

### Cannot Connect to ALB

1. Check security group rules
2. Verify target health
3. Check ECS tasks are running
4. Test health endpoints

### High Costs

1. Reduce ECS instance count
2. Remove NAT Gateway (if not needed)
3. Reduce log retention
4. Use smaller instance types

See **[PRODUCTION_DEPLOYMENT.md](PRODUCTION_DEPLOYMENT.md)** for detailed troubleshooting.

## Teardown

To destroy all infrastructure:

```bash
cd terraform
terraform destroy
```

⚠️ This will delete:
- All ECS services and tasks
- RDS database (snapshots may remain)
- Redis cluster
- Load balancer
- VPC and networking

## Security Checklist

Before production:
- [ ] Change default database password
- [ ] Enable HTTPS on ALB (ACM certificate)
- [ ] Use AWS Secrets Manager for secrets
- [ ] Enable RDS Multi-AZ
- [ ] Set up CloudWatch alarms
- [ ] Enable VPC Flow Logs
- [ ] Configure AWS WAF on ALB
- [ ] Review security groups
- [ ] Enable deletion protection on RDS
- [ ] Set up AWS Backup

## Support and Resources

- **Documentation**: See `docs/` directory
- **Terraform**: See `terraform/README.md`
- **Architecture**: See `docs/architecture/`
- **Memory Bank**: See `memory-bank/` for design decisions

## Next Steps

After deployment:
1. ✅ Verify health endpoints
2. ✅ Create test data
3. ✅ Deploy UI (rule-service-ui) - GitHub Pages
4. ✅ Load test - See [PERFORMANCE_SCALING.md](PERFORMANCE_SCALING.md)
5. ⚠️ Set up monitoring and alerts
6. ⚠️ Configure custom domain
7. ⚠️ Enable HTTPS
8. ⚠️ Document runbooks

---

**Questions?** Check the troubleshooting sections in PRODUCTION_DEPLOYMENT.md
