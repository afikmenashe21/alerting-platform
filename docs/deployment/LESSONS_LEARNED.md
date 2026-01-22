# Production Deployment - Lessons Learned

**Date**: 2026-01-22

This document captures key lessons learned during the initial AWS ECS deployment.

## Issue 1: Kafka-Zookeeper Connection Failures

### Problem
Kafka could not connect to Zookeeper on `localhost:2181` despite both containers running on the same EC2 instance with host networking.

**Error**: `java.net.ConnectException: Connection refused`

### Root Cause
- Kafka and Zookeeper were in separate ECS tasks
- Host networking mode doesn't guarantee localhost communication between separate tasks
- Startup timing issue: Kafka tried to connect before Zookeeper was fully ready

### Solution
Restarted services in correct order with delays:
1. Restart Zookeeper (wait 45s)
2. Restart Kafka (wait 30s for connection)
3. Restart dependent services

**Script**: `scripts/deployment/fix-kafka-zookeeper.sh`

### Long-term Recommendation
Combine Kafka and Zookeeper into a single ECS task definition with two containers:
- Guarantees startup ordering (`dependsOn` in container definitions)
- Ensures localhost communication works
- Reduces memory overhead
- Eliminates this entire class of failures

## Issue 2: Deployment Loop (Multiple Tasks)

### Problem
rule-service stuck with 2 tasks running (desired: 1), deployment perpetually IN_PROGRESS.

**Symptoms**:
- Service keeps starting new tasks
- Old tasks won't stop
- Deployment never completes
- FailedTasks count increasing

### Root Cause
Docker health checks were failing:
```hcl
healthCheck = {
  command = ["CMD-SHELL", "wget --quiet --tries=1 --spider http://localhost:8081/health || exit 1"]
}
```

Alpine-based Docker images only installed `ca-certificates` and `tzdata`, but **not `wget`**:
```dockerfile
RUN apk --no-cache add ca-certificates tzdata  # ← No wget!
```

**What happened**:
1. Task starts, health check fails (no wget)
2. ECS considers task UNHEALTHY
3. ECS starts replacement task
4. New task also fails health check
5. ECS won't stop old task until new one is healthy
6. Loop continues forever

### Solution
Removed health checks from task definitions:

**Terraform change** (`terraform/modules/ecs-service/main.tf`):
```hcl
# Before:
healthCheck = var.container_port > 0 ? {
  command = ["CMD-SHELL", "wget ..."]
  ...
} : null

# After:
# Health checks disabled - not needed without ALB
healthCheck = null
```

**Why this is acceptable**:
- No ALB means no target health checks needed
- Services have internal health monitoring
- ECS will restart tasks if they exit/crash
- Can re-enable later after adding `wget` to Dockerfiles

### Alternative Solutions

**Option 1: Add wget to Dockerfiles** (better long-term):
```dockerfile
RUN apk --no-cache add ca-certificates tzdata wget
```
Then rebuild and push all images.

**Option 2: Use curl instead of wget**:
```dockerfile
RUN apk --no-cache add ca-certificates tzdata curl
```
Update health check:
```hcl
command = ["CMD-SHELL", "curl -f http://localhost:${var.container_port}/health || exit 1"]
```

## Issue 3: No ALB (By Design)

### Decision
Disabled Application Load Balancer to reduce costs.

**Cost Impact**:
- ALB: ~$16/month (base) + data transfer
- Without ALB: $0

**Trade-offs**:
- ✅ Saves ~$16-20/month
- ✅ Simpler architecture
- ❌ No HTTPS termination
- ❌ No custom domain routing
- ❌ Services accessed via EC2 IPs only
- ❌ No automatic health checks

**Acceptable for**:
- Development/staging environments
- Internal services
- MVP/proof-of-concept
- Cost-sensitive deployments

**Not recommended for**:
- Production customer-facing services
- Services requiring HTTPS
- Services needing custom domains

## Key Learnings

### 1. Host Networking Limitations
- Host networking doesn't guarantee localhost communication between separate ECS tasks
- Even on the same EC2 instance, separate tasks may not connect reliably via localhost
- **Best practice**: Combine tightly-coupled services (like Kafka+Zookeeper) into single task

### 2. Docker Health Checks
- Verify health check commands work in your Docker images
- Test locally: `docker run <image> /bin/sh -c "wget ..."`
- Alpine Linux is minimal - don't assume standard tools are available
- Health checks without ALB are optional overhead

### 3. ECS Deployment Behavior
- Failed health checks cause deployment loops
- `minimumHealthyPercent=0` doesn't help if new tasks also fail
- Monitor `FailedTasks` metric to detect loops early
- Circuit breaker (if enabled) would have helped

### 4. Cost Optimization
- ALB is expensive relative to small EC2 instances
- For internal/dev services, direct EC2 access is fine
- Health checks add complexity - only use if needed
- t3.small can run entire stack (~$30/month)

### 5. Terraform State Management
- Keep state file secure (not in git)
- Use remote backend (S3) for team collaboration
- Certificate errors can break terraform operations
- Consider terraform workspaces for multiple environments

## Recommendations

### For Production Deployments

1. **Enable ALB** when budget allows:
   - HTTPS termination
   - Custom domain support
   - Better health checking
   - Blue/green deployments

2. **Add monitoring**:
   - CloudWatch alarms for unhealthy tasks
   - Failed health check alerts
   - Consumer lag monitoring
   - Error rate tracking

3. **Improve health checks**:
   - Add `wget` or `curl` to Dockerfiles
   - Test health check endpoints locally
   - Add database connectivity checks
   - Verify Kafka connection in health endpoint

4. **Use RDS Multi-AZ**:
   - Current: Single AZ (cost savings)
   - Production: Multi-AZ for HA

5. **Implement circuit breaker**:
   ```hcl
   deployment_circuit_breaker {
     enable   = true
     rollback = true
   }
   ```

### For Development

1. **Keep costs low**:
   - No ALB ✓
   - Single instance per service ✓
   - No health checks ✓
   - t3.small instances ✓

2. **Use service discovery**:
   - Already implemented for Kafka
   - Enables service-to-service communication
   - Works without ALB

3. **Local testing first**:
   - Test docker-compose setup
   - Verify health check commands locally
   - Test startup ordering

## Scripts Created

1. **fix-kafka-zookeeper.sh** - Restart services in correct order
2. **kafka-topics-commands.sh** - Guide for creating Kafka topics
3. **build-and-push.sh** - Build and push Docker images
4. **update-services.sh** - Force new ECS deployments
5. **run-migration.sh** - Run database migrations

## Documentation Created

1. **CURRENT_STATUS.md** - Current deployment status
2. **LESSONS_LEARNED.md** - This document
3. **Memory Bank updates** - Captured all decisions and issues

## Next Time

### Before Deployment
- [ ] Test health check commands in local Docker containers
- [ ] Verify all Alpine packages needed are installed
- [ ] Plan Kafka+Zookeeper as single task from start
- [ ] Decide on ALB vs no-ALB upfront

### During Deployment
- [ ] Monitor task health immediately
- [ ] Check for deployment loops early
- [ ] Keep fix scripts ready
- [ ] Document issues as they occur

### After Deployment
- [ ] Create Kafka topics before enabling services
- [ ] Test end-to-end flow
- [ ] Set up basic monitoring
- [ ] Document current state

## References

- AWS ECS Best Practices: https://docs.aws.amazon.com/AmazonECS/latest/bestpracticesguide/
- Kafka on ECS: https://aws.amazon.com/blogs/containers/deploy-apache-kafka-on-amazon-ecs/
- Alpine Docker Images: https://wiki.alpinelinux.org/wiki/Alpine_Linux_package_management
