# Production Deployment Guide - AWS ECS

This guide walks you through deploying the alerting platform to AWS using Terraform and ECS.

## Architecture Overview

The platform uses:
- **ECS with EC2 launch type** - Container orchestration on EC2 instances
- **Application Load Balancer (ALB)** - Load balancing for rule-service and alert-producer APIs
- **RDS Postgres** - Managed database (db.t3.micro free tier eligible)
- **ElastiCache Redis** - Managed cache (cache.t3.micro free tier eligible)
- **Kafka on ECS** - Self-hosted Kafka (MSK is too expensive)
- **ECR** - Container image registry
- **VPC** - Isolated network with public/private subnets
- **Auto Scaling** - Automatic scaling based on CPU/memory

## Prerequisites

1. **AWS Account** with admin access
2. **AWS CLI** installed and configured
3. **Terraform** >= 1.5.0 installed
4. **Docker** installed
5. **Make** installed (optional, for convenience)

### Install Tools

```bash
# AWS CLI (macOS)
brew install awscli
aws configure  # Enter your AWS credentials

# Terraform (macOS)
brew tap hashicorp/tap
brew install hashicorp/tap/terraform

# Docker (download from docker.com)
# Or use brew cask
brew install --cask docker
```

## Cost Estimates

### Free Tier (First 12 Months)
- ✅ 2x t3.micro EC2 instances (750 hours/month free)
- ✅ RDS db.t3.micro (750 hours/month free, 20GB storage)
- ✅ ElastiCache cache.t3.micro (if eligible)
- ✅ 750 hours ALB (first year)
- ❌ NAT Gateway: ~$32/month (NOT free tier - consider removing for dev)

### After Free Tier
Estimated: **$50-80/month** depending on usage

To reduce costs:
- Use 1 EC2 instance instead of 2
- Remove NAT Gateway (limits outbound internet from private subnets)
- Use smaller instance types
- Enable auto-scaling only when needed

## Step 1: Infrastructure Setup

### 1.1 Clone and Navigate

```bash
cd /path/to/alerting-platform
```

### 1.2 Set Up Terraform Variables

Create a `terraform/terraform.tfvars` file:

```hcl
# Core Configuration
project_name = "alerting-platform"
environment  = "prod"
aws_region   = "us-east-1"  # Change if needed

# Database Configuration
db_username = "postgres"
db_password = "YOUR_SECURE_PASSWORD_HERE"  # CHANGE THIS!

# Network Configuration
vpc_cidr = "10.0.0.0/16"
availability_zones = ["us-east-1a", "us-east-1b"]

# ECS Configuration
ecs_instance_type   = "t3.micro"      # Free tier eligible
ecs_desired_capacity = 2              # 2 for HA, 1 to save costs
ecs_min_size        = 1
ecs_max_size        = 4

# Service Configuration
service_desired_count = 1              # Start with 1 instance per service
service_max_count     = 3              # Can scale to 3

# Container Configuration
container_cpu    = 256                 # 0.25 vCPU
container_memory = 512                 # 512 MB

# Monitoring
enable_container_insights = false      # Costs money
log_retention_days       = 7           # Minimal retention

# Tags
tags = {
  Project     = "alerting-platform"
  Environment = "prod"
  ManagedBy   = "terraform"
}
```

### 1.3 Initialize Terraform

```bash
cd terraform

# Initialize Terraform
terraform init

# Validate configuration
terraform validate

# Preview changes
terraform plan
```

### 1.4 Deploy Infrastructure

```bash
# Deploy everything
terraform apply

# Type 'yes' when prompted

# This takes ~10-15 minutes
```

**What gets created:**
- VPC with public/private subnets
- ECS cluster with EC2 instances
- RDS Postgres database
- ElastiCache Redis
- Kafka + Zookeeper on ECS
- Application Load Balancer
- ECR repositories for all 6 services
- Security groups, IAM roles, etc.

### 1.5 Save Terraform Outputs

```bash
# Save important outputs
terraform output > ../deployment-info.txt

# Get ALB DNS name
terraform output alb_dns_name

# Get ECR repository URLs
terraform output ecr_repository_urls
```

## Step 2: Build and Push Docker Images

### 2.1 Authenticate to ECR

```bash
cd ..  # Back to project root

# Get AWS account ID
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
AWS_REGION="us-east-1"  # Or your region

# Login to ECR
aws ecr get-login-password --region $AWS_REGION | \
  docker login --username AWS --password-stdin \
  ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com
```

### 2.2 Build and Push All Services

Use the provided script:

```bash
./scripts/deployment/build-and-push.sh
```

Or manually for each service:

```bash
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
AWS_REGION="us-east-1"
ECR_BASE="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/alerting-platform"

# Build from project root (important for context)
for service in rule-service rule-updater evaluator aggregator sender alert-producer; do
  echo "Building $service..."
  
  # Build image
  docker build \
    -f services/$service/Dockerfile \
    -t ${ECR_BASE}/${service}:latest \
    .
  
  # Push image
  docker push ${ECR_BASE}/${service}:latest
done
```

### 2.3 Verify Images

```bash
# List images in ECR
aws ecr describe-images \
  --repository-name alerting-platform/rule-service \
  --region $AWS_REGION
```

## Step 3: Run Database Migrations

### 3.1 Connect to RDS

Get RDS endpoint from Terraform:

```bash
cd terraform
terraform output rds_endpoint
# Output: alerting-platform-prod-postgres.xxxxx.us-east-1.rds.amazonaws.com:5432
```

### 3.2 Run Migrations

Option 1: **From local machine (if NAT Gateway enabled)**

```bash
cd ..
export POSTGRES_DSN="postgres://postgres:YOUR_PASSWORD@YOUR_RDS_ENDPOINT/alerting?sslmode=require"

# Run centralized migration script
./scripts/migrations/run-migrations.sh
```

Option 2: **From EC2 instance in VPC (recommended)**

```bash
# SSH into one of the ECS EC2 instances
# Or create a temporary bastion host

# Install golang-migrate
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin/

# Run migrations
export POSTGRES_DSN="postgres://postgres:YOUR_PASSWORD@YOUR_RDS_ENDPOINT/alerting?sslmode=require"
migrate -path /path/to/migrations -database $POSTGRES_DSN up
```

Option 3: **Run migrations in ECS task**

Create a one-off ECS task that runs migrations on startup.

## Step 4: Create Kafka Topics

### 4.1 Get Kafka Endpoint

```bash
cd terraform
terraform output kafka_endpoint
# Output: kafka.alerting-platform-prod.local:9092
```

### 4.2 Create Topics

Connect to ECS cluster and run topic creation:

```bash
# Find Kafka task
aws ecs list-tasks --cluster alerting-platform-prod-cluster --region us-east-1

# Execute command in Kafka container
aws ecs execute-command \
  --cluster alerting-platform-prod-cluster \
  --task <KAFKA_TASK_ARN> \
  --container kafka \
  --interactive \
  --command "/bin/bash"

# Inside container, create topics
kafka-topics --create --bootstrap-server localhost:9092 \
  --topic alerts.new --partitions 9 --replication-factor 1
kafka-topics --create --bootstrap-server localhost:9092 \
  --topic rule.changed --partitions 9 --replication-factor 1
kafka-topics --create --bootstrap-server localhost:9092 \
  --topic alerts.matched --partitions 9 --replication-factor 1
kafka-topics --create --bootstrap-server localhost:9092 \
  --topic notifications.ready --partitions 9 --replication-factor 1
```

## Step 5: Deploy Services

### 5.1 Update ECS Services

Services are already created by Terraform. Force new deployment to use latest images:

```bash
# Update all services
for service in rule-service rule-updater evaluator aggregator sender alert-producer; do
  aws ecs update-service \
    --cluster alerting-platform-prod-cluster \
    --service $service \
    --force-new-deployment \
    --region us-east-1
done
```

### 5.2 Monitor Deployments

```bash
# Check service status
aws ecs describe-services \
  --cluster alerting-platform-prod-cluster \
  --services rule-service evaluator aggregator sender rule-updater alert-producer \
  --region us-east-1 \
  --query 'services[*].[serviceName,runningCount,desiredCount,deployments[0].status]' \
  --output table

# View service logs
aws logs tail /ecs/alerting-platform/prod/rule-service --follow --region us-east-1
```

## Step 6: Verify Deployment

### 6.1 Get Service URLs

```bash
cd terraform
ALB_DNS=$(terraform output -raw alb_dns_name)

echo "Rule Service API: http://${ALB_DNS}:8080"
echo "Alert Producer API: http://${ALB_DNS}:8081"
```

### 6.2 Test Health Endpoints

```bash
# Test rule-service
curl http://${ALB_DNS}:8080/health

# Test alert-producer
curl http://${ALB_DNS}:8081/health

# List clients (should be empty initially)
curl http://${ALB_DNS}:8080/api/v1/clients
```

### 6.3 Create Test Data

```bash
# Create a client
curl -X POST http://${ALB_DNS}:8080/api/v1/clients \
  -H "Content-Type: application/json" \
  -d '{"name": "Test Client", "email": "test@example.com"}'

# Create a rule (get client_id from previous response)
curl -X POST http://${ALB_DNS}:8080/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "YOUR_CLIENT_ID",
    "severity": "HIGH",
    "source": "*",
    "name": "*"
  }'
```

## Step 7: Continuous Deployment (CI/CD)

The platform includes GitHub Actions workflow for automated deployments.

### 7.1 Configure GitHub Secrets

In your GitHub repository, add these secrets:

```
AWS_ACCESS_KEY_ID       - AWS access key
AWS_SECRET_ACCESS_KEY   - AWS secret key
AWS_REGION              - us-east-1 (or your region)
ECR_REPOSITORY_PREFIX   - alerting-platform
DB_PASSWORD             - Your RDS password
```

### 7.2 Deploy on Push

The workflow automatically:
1. Builds Docker images on push to `main` branch
2. Pushes to ECR
3. Updates ECS services

See `.github/workflows/deploy.yml` for details.

## Scaling Services

### Manual Scaling

```bash
# Scale evaluator to 3 instances
aws ecs update-service \
  --cluster alerting-platform-prod-cluster \
  --service evaluator \
  --desired-count 3 \
  --region us-east-1
```

### Auto Scaling

Auto-scaling is configured via Terraform:
- **CPU target**: 70%
- **Memory target**: 80%
- **Min instances**: 1 (service_desired_count)
- **Max instances**: 3 (service_max_count)

Services will automatically scale based on load.

**Note**: rule-updater MUST stay at 1 instance (never scale it).

## Monitoring

### CloudWatch Logs

```bash
# View logs
aws logs tail /ecs/alerting-platform/prod/evaluator --follow --region us-east-1

# Search logs
aws logs filter-log-events \
  --log-group-name /ecs/alerting-platform/prod/evaluator \
  --filter-pattern "ERROR" \
  --region us-east-1
```

### CloudWatch Metrics

View in AWS Console:
- ECS → Clusters → alerting-platform-prod-cluster → Metrics
- Services → Individual service → Metrics

Key metrics:
- CPUUtilization
- MemoryUtilization
- TargetResponseTime (ALB)
- ActiveConnectionCount (RDS)

## Troubleshooting

### Services Not Starting

```bash
# Check task status
aws ecs describe-tasks \
  --cluster alerting-platform-prod-cluster \
  --tasks $(aws ecs list-tasks --cluster alerting-platform-prod-cluster --service-name evaluator --query 'taskArns[0]' --output text) \
  --region us-east-1

# Common issues:
# - Database connection: Check security groups
# - Kafka connection: Check service discovery
# - Out of memory: Increase container_memory in terraform.tfvars
```

### Cannot Connect to Services

```bash
# Check security groups
# Make sure ALB security group allows inbound on ports 8080, 8081
# Make sure ECS security group allows inbound from ALB

# Check target health
aws elbv2 describe-target-health \
  --target-group-arn $(aws elbv2 describe-target-groups --names alerting-platform-prod-rule-svc --query 'TargetGroups[0].TargetGroupArn' --output text) \
  --region us-east-1
```

### High Costs

```bash
# Check resource usage
aws ce get-cost-and-usage \
  --time-period Start=2026-01-01,End=2026-01-31 \
  --granularity MONTHLY \
  --metrics BlendedCost \
  --group-by Type=SERVICE

# To reduce costs:
# 1. Scale down to 1 ECS instance
# 2. Remove NAT Gateway (set create_nat_gateway = false in terraform)
# 3. Reduce log retention to 1 day
# 4. Use smaller instance types
```

## Teardown

To destroy all resources:

```bash
cd terraform

# Destroy everything
terraform destroy

# Type 'yes' when prompted

# Manual cleanup (if needed):
# - Empty ECR repositories first
# - Delete CloudWatch log groups
# - Delete RDS snapshots
```

## Production Checklist

Before going to production:

- [ ] Change default database password
- [ ] Enable HTTPS on ALB (add ACM certificate)
- [ ] Enable RDS automated backups
- [ ] Enable RDS Multi-AZ for HA
- [ ] Set up CloudWatch alarms
- [ ] Configure log aggregation (CloudWatch Insights or external)
- [ ] Enable AWS Config for compliance
- [ ] Set up AWS Backup for RDS
- [ ] Review security groups (principle of least privilege)
- [ ] Enable VPC Flow Logs
- [ ] Set up AWS WAF on ALB
- [ ] Configure Route 53 for custom domain
- [ ] Enable deletion protection on RDS
- [ ] Document runbooks for incidents
- [ ] Set up on-call rotation
- [ ] Load test the platform

## Next Steps

1. **Set up monitoring dashboard** - CloudWatch or Grafana
2. **Configure alerts** - PagerDuty, OpsGenie, or SNS
3. **Deploy UI** - Deploy rule-service-ui to S3 + CloudFront
4. **Set up staging environment** - Use terraform workspaces
5. **Implement blue/green deployments** - Use ECS deployment strategies

## Support

For issues or questions:
- Check CloudWatch logs first
- Review security groups and networking
- Consult AWS ECS documentation
- Check Terraform state for resource details
