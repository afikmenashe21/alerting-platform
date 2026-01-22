# Production Deployment - Quick Start

Get your alerting platform running on AWS in under 30 minutes.

## Prerequisites

- AWS account with admin access
- AWS CLI installed and configured
- Terraform >= 1.5.0
- Docker installed
- Git repository access

## Step 1: Configure Terraform (5 min)

```bash
cd terraform

# Copy example config
cp terraform.tfvars.example terraform.tfvars

# Edit config (IMPORTANT: Set db_password!)
vim terraform.tfvars
```

Minimal config:
```hcl
db_password = "YourSecurePassword123!"  # CHANGE THIS
```

## Step 2: Deploy Infrastructure (10-15 min)

```bash
# Initialize
terraform init

# Deploy
terraform apply
# Type 'yes' when prompted

# Save outputs
terraform output > ../deployment-info.txt
terraform output -raw alb_dns_name > ../alb-dns.txt
```

## Step 3: Build and Deploy Services (10 min)

```bash
cd ..

# Build and push all Docker images
./scripts/deployment/build-and-push.sh

# Update ECS services
./scripts/deployment/update-services.sh
```

## Step 4: Initialize Database (5 min)

```bash
# Get RDS endpoint
RDS_ENDPOINT=$(cd terraform && terraform output -raw rds_endpoint)

# Run migrations (from EC2 instance in VPC, or configure bastion)
export POSTGRES_DSN="postgres://postgres:YourPassword@${RDS_ENDPOINT}/alerting?sslmode=require"
./scripts/migrations/run-migrations.sh
```

## Step 5: Create Kafka Topics (2 min)

```bash
# Connect to Kafka task and create topics
aws ecs execute-command \
  --cluster alerting-platform-prod-cluster \
  --task $(aws ecs list-tasks --cluster alerting-platform-prod-cluster --service-name kafka --query 'taskArns[0]' --output text) \
  --container kafka \
  --interactive \
  --command "/bin/bash"

# Inside container:
for topic in alerts.new rule.changed alerts.matched notifications.ready; do
  kafka-topics --create --bootstrap-server localhost:9092 \
    --topic $topic --partitions 9 --replication-factor 1
done
exit
```

## Step 6: Verify (2 min)

```bash
ALB_DNS=$(cat alb-dns.txt)

# Test health endpoints
curl http://${ALB_DNS}:8080/health  # rule-service
curl http://${ALB_DNS}:8081/health  # alert-producer

# Create test client
curl -X POST http://${ALB_DNS}:8080/api/v1/clients \
  -H "Content-Type: application/json" \
  -d '{"name": "Test Client", "email": "test@example.com"}'
```

✅ **Done!** Your alerting platform is running on AWS.

## What You Deployed

- 2 EC2 instances (ECS cluster)
- 6 containerized services
- RDS Postgres database
- ElastiCache Redis
- Kafka on ECS
- Application Load Balancer

## Next Steps

1. **Set up CI/CD**: Configure GitHub Actions with AWS credentials
2. **Configure monitoring**: Set up CloudWatch alarms
3. **Enable HTTPS**: Add ACM certificate to ALB
4. **Deploy UI**: Deploy rule-service-ui to S3 + CloudFront

## Costs

- **Free tier**: $35-50/month (mainly NAT Gateway)
- **After free tier**: $100-110/month

## Troubleshooting

**Services not starting?**
```bash
aws logs tail /ecs/alerting-platform/prod/<service-name> --follow
```

**Can't connect to ALB?**
```bash
# Check target health
aws elbv2 describe-target-health \
  --target-group-arn $(aws elbv2 describe-target-groups --names alerting-platform-prod-rule-svc --query 'TargetGroups[0].TargetGroupArn' --output text)
```

**Need to destroy everything?**
```bash
cd terraform
terraform destroy
```

## Full Documentation

See `PRODUCTION_DEPLOYMENT.md` for complete guide.
