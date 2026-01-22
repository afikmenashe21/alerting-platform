# Ultra-Low-Cost Deployment Guide

Get your alerting platform running on AWS for **~$5-10/month** (or FREE with free tier).

## Cost Breakdown

### With AWS Free Tier (First 12 Months)
| Resource | Cost |
|----------|------|
| EC2 (1x t3.micro) | **$0** (750 hrs/month free) |
| RDS (db.t3.micro) | **$0** (750 hrs/month free) |
| ElastiCache (cache.t3.micro) | **$0** (if free tier available) |
| ALB | **$0** (750 hrs first year) |
| NAT Gateway | **$0** (removed!) |
| Data transfer | ~$0-5 (minimal) |
| CloudWatch Logs | ~$0-2 (1 day retention) |
| **TOTAL** | **~$0-10/month** ✅ |

### After Free Tier
| Resource | Cost |
|----------|------|
| EC2 (1x t3.micro) | ~$7/month |
| RDS (db.t3.micro) | ~$15/month |
| ElastiCache | ~$12/month |
| ALB | ~$18/month |
| Data transfer | ~$5/month |
| **TOTAL** | **~$57/month** |

## What Changed to Save $40/month?

### 1. ❌ Removed NAT Gateway (Saves $32/month!)

**Impact**: ECS tasks in private subnets can't reach the internet directly.

**Workaround**: 
- ECR uses VPC endpoints (no internet needed)
- Services communicate internally
- For external APIs (SMTP, Slack, webhooks), use VPC endpoints or move to public subnet

### 2. 📉 Single EC2 Instance (Saves $7/month)

Changed from 2 instances to 1.

**Impact**: No high availability, but fine for dev/low-traffic.

### 3. 📉 No Auto-Scaling (Saves on potential costs)

Services stay at 1 instance each.

**Impact**: Can't handle traffic spikes automatically, but you can still manually scale.

### 4. 📉 Smaller Containers (More efficient)

Reduced CPU/memory per container.

**Impact**: 6 services fit on 1 t3.micro instance.

### 5. 📉 Reduced Log Retention (Saves $1-2/month)

Logs kept for 1 day instead of 7.

**Impact**: Less historical logs, but CloudWatch still works.

## Ultra-Low-Cost Configuration

Use this configuration file:

```bash
cp terraform/terraform.tfvars.ultra-low-cost terraform/terraform.tfvars
vim terraform/terraform.tfvars  # Change db_password
```

Key settings:
```hcl
ecs_desired_capacity = 1       # Just 1 EC2 instance
service_max_count = 1          # No auto-scaling
container_cpu = 128            # Smaller containers
container_memory = 256         # Smaller containers
log_retention_days = 1         # Minimal logs
```

## NAT Gateway Alternatives

Since we removed NAT Gateway, here are options for outbound internet:

### Option 1: VPC Endpoints (Recommended)

Use AWS VPC endpoints for AWS services:

```hcl
# Add to terraform/modules/vpc/main.tf

# ECR endpoints (for pulling images)
resource "aws_vpc_endpoint" "ecr_api" {
  vpc_id            = aws_vpc.main.id
  service_name      = "com.amazonaws.${data.aws_region.current.name}.ecr.api"
  vpc_endpoint_type = "Interface"
  subnet_ids        = aws_subnet.private[*].id
  security_group_ids = [aws_security_group.vpc_endpoints.id]
}

resource "aws_vpc_endpoint" "ecr_dkr" {
  vpc_id            = aws_vpc.main.id
  service_name      = "com.amazonaws.${data.aws_region.current.name}.ecr.dkr"
  vpc_endpoint_type = "Interface"
  subnet_ids        = aws_subnet.private[*].id
  security_group_ids = [aws_security_group.vpc_endpoints.id]
}

# S3 endpoint (for ECR layers)
resource "aws_vpc_endpoint" "s3" {
  vpc_id       = aws_vpc.main.id
  service_name = "com.amazonaws.${data.aws_region.current.name}.s3"
  route_table_ids = [aws_route_table.private.id]
}
```

**Cost**: ~$7-10/month (cheaper than NAT Gateway)

### Option 2: Public Subnet for Sender Service

Move the `sender` service (which needs SMTP/webhooks) to public subnet:

```hcl
# In terraform/main.tf for sender service
module "sender" {
  # ... existing config ...
  
  # Use public subnets instead of private
  private_subnet_ids = module.vpc.public_subnet_ids  # Changed!
  
  # Assign public IP
  # (modify ecs-service module to support this)
}
```

**Cost**: Free! But less secure.

### Option 3: Re-enable NAT Gateway When Needed

You can enable it temporarily:

```hcl
# In terraform.tfvars
# Set this in modules/vpc call:
create_nat_gateway = true
```

Then `terraform apply` to add it, use it, then remove it.

## Deployment with Ultra-Low-Cost Config

```bash
# 1. Use ultra-low-cost config
cd terraform
cp terraform.tfvars.ultra-low-cost terraform.tfvars
vim terraform.tfvars  # Set db_password

# 2. Deploy
terraform init
terraform apply

# 3. Build and deploy services
cd ..
./scripts/deployment/build-and-push.sh
./scripts/deployment/update-services.sh
```

## Limitations of Ultra-Low-Cost Setup

⚠️ **What you lose:**

1. **No high availability** - Single EC2 instance
2. **No auto-scaling** - Manual scaling only
3. **Limited capacity** - All 6 services + Kafka on 1 t3.micro
4. **No outbound internet** - Unless you add VPC endpoints
5. **Slower recovery** - Restarts take longer with 1 instance
6. **Limited logs** - Only 1 day retention

✅ **What you keep:**

1. All services running
2. Load balancer
3. Managed database (RDS)
4. Managed cache (Redis)
5. Kafka messaging
6. Health checks
7. CloudWatch monitoring

## When to Use This

- ✅ **Development/Testing**: Perfect for dev environments
- ✅ **Low-traffic production**: <100 alerts/hour
- ✅ **Proof of concept**: Demonstrating the platform
- ✅ **Learning**: Experimenting with AWS

- ❌ **High-traffic production**: Use standard config
- ❌ **Mission-critical**: Need HA and auto-scaling
- ❌ **Heavy load**: Need more resources

## Upgrading from Ultra-Low-Cost

When you need more capacity:

```hcl
# In terraform.tfvars

# Add HA
ecs_desired_capacity = 2

# Enable auto-scaling
service_max_count = 3

# Larger containers
container_cpu = 256
container_memory = 512

# Better logging
log_retention_days = 7
```

Then `terraform apply` to upgrade.

## Alternative: Use AWS Lightsail

For even lower costs, consider AWS Lightsail:
- $5/month for 512MB RAM, 1 vCPU
- $10/month for 1GB RAM, 1 vCPU
- Includes fixed bandwidth
- Simpler than ECS

But requires different setup (Docker Compose on Lightsail instance).

## Free Tier Limits to Watch

AWS Free Tier limits (12 months):
- ✅ 750 hours/month EC2 t2.micro or t3.micro
- ✅ 750 hours/month RDS db.t2.micro or db.t3.micro
- ✅ 20GB RDS storage
- ✅ 750 hours/month ALB (first year)
- ✅ 15GB data transfer out/month

⚠️ **If you exceed these**, you'll be charged standard rates.

## Cost Monitoring

Set up billing alerts:

```bash
# Set up AWS Budget
aws budgets create-budget \
  --account-id YOUR_ACCOUNT_ID \
  --budget file://budget.json

# budget.json:
{
  "BudgetName": "monthly-budget",
  "BudgetLimit": {
    "Amount": "10",
    "Unit": "USD"
  },
  "TimeUnit": "MONTHLY",
  "BudgetType": "COST"
}
```

## Summary

**Standard Config**: $50/month (free tier) → $110/month (after)  
**Ultra-Low-Cost**: $5-10/month (free tier) → $57/month (after)  
**Savings**: $40/month! 💰

Main tradeoff: No NAT Gateway means limited outbound internet access.

---

**Ready to deploy?** Use `terraform.tfvars.ultra-low-cost` and follow QUICKSTART.md!
