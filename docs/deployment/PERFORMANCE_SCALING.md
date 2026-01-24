# Performance Scaling Guide

This document describes the performance characteristics, scaling strategies, and load test results for the alerting platform.

## Current Configuration (Ultra-Low-Cost)

| Resource | Configuration | Cost |
|----------|--------------|------|
| EC2 Instance | 1x t3.small (2 vCPU, 2GB RAM) | ~$15/month |
| Container Memory | 150 MB per service | - |
| Kafka | 1 broker, 6 partitions per topic | - |
| RDS | db.t3.micro (free tier) | $0 |
| ElastiCache | cache.t3.micro (free tier) | $0 |

### Task Distribution

| Service | Instances | Memory Each | Total Memory |
|---------|-----------|-------------|--------------|
| Kafka + Zookeeper | 1 | 576 MB | 576 MB |
| Evaluator | 2 | 150 MB | 300 MB |
| Aggregator | 2 | 150 MB | 300 MB |
| Sender | 1 | 150 MB | 150 MB |
| Rule Service | 1 | 150 MB | 150 MB |
| Rule Updater | 1 | 150 MB | 150 MB |
| Metrics Service | 1 | 150 MB | 150 MB |
| Alert Producer | 1 | 150 MB | 150 MB |
| **Total** | **10** | - | **~1926 MB** |

> Note: t3.small has ~1938 MB usable after ECS agent overhead

## Load Test Results

### Test: 100,000 Alerts Burst

```
Configuration:
- Kafka partitions: 6
- Evaluator instances: 2
- Aggregator instances: 2
- Sender instances: 1 (rate limited)

Results:
- Duration: 151 seconds
- Total alerts: 100,000
- Errors: 0
```

| Component | Rate (sustained) | Latency | Errors |
|-----------|------------------|---------|--------|
| Alert Producer | ~780/s | - | 0 |
| Evaluator | ~320/s | 3.2ms | 0 |
| Aggregator | ~100/s | 5.0ms | 0 |
| Sender | ~120/s | - | 0 |

### Bottlenecks Identified

1. **Kafka Single Broker**: Write throughput limited to ~800 alerts/sec
2. **Memory Constraint**: Only ~38 MB free after all services running
3. **Pipeline Rate**: Evaluator processes faster than aggregator creates DB records

## Scaling Strategies

### Free Optimizations (No Additional Cost)

#### 1. Increase Kafka Partitions

Kafka auto-creates topics with `KAFKA_NUM_PARTITIONS` setting:

```hcl
# terraform/modules/kafka/combined-task.tf
{ name = "KAFKA_NUM_PARTITIONS", value = "6" }
```

More partitions = more parallel consumers. With 6 partitions, up to 6 instances of each consumer group can process in parallel.

#### 2. Scale ECS Tasks

Increase `desired_count` in `terraform/main.tf`:

```hcl
module "evaluator" {
  desired_count = 2  # Scale to 2 instances
  max_count     = 3  # Allow auto-scaling to 3
}
```

> Note: Limited by available EC2 memory

#### 3. Reduce Container Memory

Reduce memory per container to fit more instances:

```hcl
# terraform/terraform.tfvars
container_memory = 150  # Reduced from 180 MB
```

### Paid Optimizations

| Upgrade | Benefit | Additional Cost |
|---------|---------|-----------------|
| t3.medium | 2x memory (4GB) | +~$15/month |
| t3.large | 4x memory (8GB) | +~$45/month |
| 2nd Kafka broker | 2x write throughput | Container resources |
| RDS upgrade | More connections | $15-30/month |

## Configuration Reference

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `KAFKA_NUM_PARTITIONS` | Partitions for auto-created topics | 6 |
| `EMAIL_RATE_LIMIT` | Email sends per second | 2 |
| `EMAIL_PROVIDER` | Force provider (resend/ses) | auto |

### Terraform Variables

```hcl
# terraform/terraform.tfvars

# Service scaling
service_desired_count = 1   # Default task count
service_max_count     = 2   # Max for auto-scaling

# Container resources
container_cpu    = 128      # 0.125 vCPU
container_memory = 150      # 150 MB

# Kafka
kafka_partitions = 6        # Per topic
```

## Monitoring Performance

### Metrics API

```bash
# Get all service metrics
curl https://<api-gateway>/metrics-api/api/v1/services/metrics

# Response includes:
# - messages_per_second (real-time rate)
# - avg_processing_latency_ns
# - messages_processed (total)
# - processing_errors
```

### CloudWatch Logs

```bash
# View evaluator throughput
aws logs filter-log-events \
  --log-group-name "/ecs/alerting-platform/prod/evaluator" \
  --filter-pattern "processed" \
  --limit 10
```

### EC2 Resource Usage

```bash
# Check container instance resources
aws ecs describe-container-instances \
  --cluster alerting-platform-prod-cluster \
  --container-instances $(aws ecs list-container-instances \
    --cluster alerting-platform-prod-cluster \
    --query 'containerInstanceArns[0]' --output text) \
  --query 'containerInstances[0].{
    remainingCPU: remainingResources[?name==`CPU`].integerValue | [0],
    remainingMemory: remainingResources[?name==`MEMORY`].integerValue | [0],
    runningTasks: runningTasksCount
  }'
```

## Email Rate Limiting

The sender service implements a token bucket rate limiter at the provider level:

```go
// services/sender/internal/sender/email/provider/provider.go
rateLimiter := make(chan struct{}, rateLimit)
// Tokens replenished at configured rate (default: 2/sec)
```

### Test Email Filtering

Emails to test domains are automatically skipped (no quota used):

- `@example.com`, `@example.org`, `@example.net`
- `@test.com`
- `@localhost`
- `@invalid`

## Applying Changes

1. **Update Terraform variables**:
   ```bash
   cd terraform
   vim terraform.tfvars
   ```

2. **Apply infrastructure changes**:
   ```bash
   terraform plan -out=tfplan
   terraform apply tfplan
   ```

3. **Force service redeployment** (for Kafka partition changes):
   ```bash
   aws ecs update-service --cluster alerting-platform-prod-cluster \
     --service kafka-combined --force-new-deployment
   
   # Wait, then redeploy consumers
   for svc in evaluator aggregator sender; do
     aws ecs update-service --cluster alerting-platform-prod-cluster \
       --service $svc --force-new-deployment
   done
   ```

## Capacity Planning

| Throughput Target | Required Changes |
|-------------------|------------------|
| 500/s | Current config (t3.small, 6 partitions) |
| 1000/s | t3.medium + 3 evaluator instances |
| 2000/s | t3.large + 2 Kafka brokers + 6 instances |
| 5000/s | Multiple EC2 + MSK (managed Kafka) |

## Related Documentation

- [Ultra Low Cost Deployment](ULTRA_LOW_COST.md)
- [Production Deployment](PRODUCTION_DEPLOYMENT.md)
- [System Patterns](../../memory-bank/systemPatterns.md)
