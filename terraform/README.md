# Terraform Infrastructure for Alerting Platform

This directory contains Terraform configuration for deploying the alerting platform to AWS ECS.

## Quick Start

```bash
# 1. Copy example vars file
cp terraform.tfvars.example terraform.tfvars

# 2. Edit terraform.tfvars and set your values
vim terraform.tfvars  # IMPORTANT: Set db_password!

# 3. Initialize Terraform
terraform init

# 4. Preview changes
terraform plan

# 5. Deploy infrastructure
terraform apply
```

## Architecture

The infrastructure includes:

- **VPC**: Isolated network with public/private subnets across 2 AZs
- **ECS Cluster**: Container orchestration on EC2 instances (t3.micro)
- **Application Load Balancer**: Public access to rule-service and alert-producer APIs
- **RDS Postgres**: Managed database (db.t3.micro free tier eligible)
- **ElastiCache Redis**: Managed cache (cache.t3.micro free tier eligible)
- **Kafka on ECS**: Self-hosted Kafka + Zookeeper (MSK is too expensive)
- **ECR**: Docker image registry for all services
- **Auto Scaling**: Automatic scaling based on CPU/memory utilization

## Modules

- `modules/vpc/` - VPC, subnets, NAT gateway, routing
- `modules/ecr/` - ECR repositories for Docker images
- `modules/ecs-cluster/` - ECS cluster, EC2 instances, auto-scaling
- `modules/rds/` - RDS Postgres database
- `modules/redis/` - ElastiCache Redis cluster
- `modules/kafka/` - Kafka and Zookeeper on ECS
- `modules/alb/` - Application Load Balancer and target groups
- `modules/ecs-service/` - Reusable module for ECS services

## Services

All 6 services are deployed as ECS services:

| Service | Port | Load Balanced | Scaling |
|---------|------|---------------|---------|
| rule-service | 8080 | ✅ Yes | 1-3 instances |
| alert-producer | 8081 | ✅ Yes | 1-3 instances |
| evaluator | - | ❌ No | 1-3 instances |
| aggregator | - | ❌ No | 1-3 instances |
| sender | - | ❌ No | 1-3 instances |
| rule-updater | - | ❌ No | **1 only** (never scale!) |

## Variables

Key variables in `terraform.tfvars`:

```hcl
# Required
db_password = "..."  # Set to a secure password

# Cost optimization
ecs_desired_capacity = 1   # Use 1 to save costs
service_desired_count = 1  # Start with 1 instance per service

# Scaling
service_max_count = 3      # Can scale to 3 instances
ecs_max_size = 4           # Can scale cluster to 4 EC2 instances
```

## Outputs

Important outputs after deployment:

```bash
# Get outputs
terraform output

# Key outputs:
terraform output alb_dns_name              # ALB DNS name
terraform output rule_service_url          # Rule Service API URL
terraform output alert_producer_url        # Alert Producer API URL
terraform output ecr_repository_urls       # ECR repos for all services
terraform output rds_endpoint              # RDS endpoint (sensitive)
terraform output kafka_endpoint            # Kafka bootstrap servers
```

## Cost Estimates

### Free Tier (First 12 Months)
- ✅ 2x t3.micro EC2: Free (750 hours/month)
- ✅ RDS db.t3.micro: Free (750 hours/month, 20GB)
- ✅ ElastiCache cache.t3.micro: Free (if eligible)
- ✅ ALB: Partially free (750 hours first year)
- ❌ NAT Gateway: ~$32/month (NOT free)
- ❌ Data transfer: Variable

**Estimated monthly cost**: $35-50 (mainly NAT Gateway)

### After Free Tier
- EC2 (2x t3.micro): ~$15/month
- RDS (db.t3.micro): ~$15/month
- ElastiCache: ~$12/month
- NAT Gateway: ~$32/month
- ALB: ~$18/month
- Data transfer: ~$5-10/month

**Total**: ~$100-110/month

### Cost Savings Tips

1. **Use 1 EC2 instance** instead of 2:
   ```hcl
   ecs_desired_capacity = 1
   ```

2. **Remove NAT Gateway** (if you don't need outbound internet from private subnets):
   ```hcl
   # In modules/vpc/variables.tf
   create_nat_gateway = false
   ```

3. **Reduce service count**:
   ```hcl
   service_desired_count = 1
   ```

4. **Smaller containers**:
   ```hcl
   container_cpu = 128     # 0.125 vCPU
   container_memory = 256  # 256 MB
   ```

## Remote State (Optional)

For team collaboration, store Terraform state in S3:

```hcl
# In main.tf, uncomment backend block:
backend "s3" {
  bucket         = "alerting-platform-terraform-state"
  key            = "prod/terraform.tfstate"
  region         = "us-east-1"
  dynamodb_table = "alerting-platform-terraform-locks"
  encrypt        = true
}
```

Create S3 bucket and DynamoDB table:

```bash
# Create S3 bucket
aws s3 mb s3://alerting-platform-terraform-state --region us-east-1
aws s3api put-bucket-versioning \
  --bucket alerting-platform-terraform-state \
  --versioning-configuration Status=Enabled

# Create DynamoDB table for locking
aws dynamodb create-table \
  --table-name alerting-platform-terraform-locks \
  --attribute-definitions AttributeName=LockID,AttributeType=S \
  --key-schema AttributeName=LockID,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST \
  --region us-east-1
```

## Troubleshooting

### "No space left on device" error

ECS instances running out of disk space. SSH into instance and clean Docker:

```bash
docker system prune -a -f
```

Or increase ECS instance root volume size in `modules/ecs-cluster/main.tf`.

### Services not starting

Check CloudWatch logs:

```bash
aws logs tail /ecs/alerting-platform/prod/<service-name> --follow
```

Common issues:
- Database connection failure → Check security groups
- Kafka connection failure → Check service discovery DNS
- OOM (out of memory) → Increase `container_memory`

### Cannot connect to ALB

Check:
1. Security group allows inbound on ports 8080, 8081
2. Target health is healthy: `aws elbv2 describe-target-health ...`
3. ECS tasks are running: `aws ecs list-tasks ...`

## Cleanup

To destroy all infrastructure:

```bash
# Preview what will be destroyed
terraform plan -destroy

# Destroy everything
terraform destroy

# Manual cleanup if needed:
# 1. Empty ECR repositories
# 2. Delete CloudWatch log groups
# 3. Delete RDS snapshots
```

## Security Notes

⚠️ **Important Security Considerations:**

1. **Never commit `terraform.tfvars`** - Contains sensitive data
2. **Use AWS Secrets Manager** for production passwords
3. **Enable HTTPS** on ALB (requires ACM certificate)
4. **Review security groups** - Follow principle of least privilege
5. **Enable VPC Flow Logs** for network monitoring
6. **Enable RDS encryption at rest** (already configured)
7. **Use IAM roles** instead of access keys where possible
8. **Enable AWS Config** for compliance monitoring
9. **Set up AWS GuardDuty** for threat detection
10. **Regular security audits** with AWS Security Hub

## Next Steps

After deploying infrastructure:

1. Build and push Docker images: `../scripts/deployment/build-and-push.sh`
2. Run database migrations (see deployment guide)
3. Create Kafka topics (see deployment guide)
4. Update ECS services: `../scripts/deployment/update-services.sh`
5. Verify deployment: `curl http://$(terraform output -raw alb_dns_name):8080/health`

See `../docs/deployment/PRODUCTION_DEPLOYMENT.md` for complete guide.
