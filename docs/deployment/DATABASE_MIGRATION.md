# Database Migration - Completed

## Overview

Successfully completed the initial database schema migration for the alerting platform on AWS RDS Postgres.

**Date**: January 21, 2026  
**Status**: ✅ Completed  
**Approach**: Docker-based ECS one-off task

## Migration Details

### Tables Created

All 4 required tables were successfully created:

1. **clients** - Client/tenant information
2. **rules** - Alert matching rules
3. **endpoints** - Notification endpoints (email, Slack, webhook)
4. **notifications** - Notification records (idempotency boundary)

### Migration Approach

We used a Docker-based approach that runs as a one-off ECS task:

```
┌─────────────────────┐
│ Migration Docker    │
│ Image (postgres:15) │
│ + init-schema.sql   │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ Push to ECR         │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ Run as ECS Task     │
│ (bridge network)    │
│ DB credentials via  │
│ environment vars    │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ RDS Postgres        │
│ (private subnet)    │
└─────────────────────┘
```

### Why This Approach?

1. **Repeatable**: Dockerized migration can run on any ECS cluster
2. **Secure**: DB credentials passed via environment variables, not hardcoded
3. **Network Access**: ECS tasks in same VPC can reach RDS in private subnet
4. **Auditable**: Complete logs in CloudWatch Logs
5. **No SSH Required**: No need for bastion hosts or VPN

## Files Created

### Migration Docker Image

- **Dockerfile**: `migrations/Dockerfile`
  - Based on `postgres:15-alpine`
  - Includes SQL migration file
  - Includes entrypoint script with DB connectivity checks

- **Entrypoint**: `migrations/docker-entrypoint.sh`
  - Waits for database to be ready
  - Runs migration SQL
  - Verifies tables were created
  - Exits with proper status codes

- **SQL Schema**: `migrations/init-schema.sql`
  - Complete schema with all tables
  - Indexes for performance
  - Foreign key constraints
  - Wildcard support (nullable fields)

### Deployment Scripts

- **Main Script**: `scripts/deployment/run-migration.sh`
  - Builds Docker image
  - Pushes to ECR
  - Registers ECS task definition
  - Runs task and waits for completion
  - Shows CloudWatch logs location

- **SSM Alternative**: `scripts/deployment/run-migrations-ssm-simple.sh`
  - Alternative approach using SSM (Systems Manager)
  - Requires SSM agent on EC2 instances
  - Note: We added SSM policy to Terraform for future use

- **Manual Guide**: `scripts/deployment/MIGRATION_GUIDE.md`
  - Comprehensive guide with multiple approaches
  - Troubleshooting steps
  - Verification commands

## Infrastructure Updates

### Terraform Changes

Added SSM policy to ECS instance IAM role for future administrative access:

```hcl
resource "aws_iam_role_policy_attachment" "ecs_instance_ssm" {
  role       = aws_iam_role.ecs_instance.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}
```

**File**: `terraform/modules/ecs-cluster/main.tf`

This enables:
- AWS Systems Manager Session Manager access
- Remote command execution via `aws ssm send-command`
- Future administrative tasks without SSH

## Execution Log

```
[INFO] Creating ECR repository...
[✓] ECR Repository: 248508119478.dkr.ecr.us-east-1.amazonaws.com/alerting-platform-prod-migration

[INFO] Building migration Docker image...
[✓] Image built

[INFO] Pushing to ECR...
[✓] Image pushed to ECR

[INFO] Registering ECS task definition...
[✓] Task definition: arn:aws:ecs:us-east-1:248508119478:task-definition/alerting-platform-prod-migration:1

[INFO] Running migration task...
[✓] Task started

[INFO] Waiting for migration to complete...
[✓] Migration completed successfully!
```

### Database Logs

```
================================
Database Migration Runner
================================
Connecting to: alerting-platform-prod-postgres.cot8kqgoccg6.us-east-1.rds.amazonaws.com:5432/alerting
User: postgres

Waiting for database to be ready...
✓ Database is ready

Running migrations...
================================
CREATE TABLE   (clients)
CREATE TABLE   (rules)
CREATE TABLE   (endpoints)
ALTER TABLE    (drop old email columns)
CREATE TABLE   (notifications)
CREATE INDEX   (notifications performance indexes)
ALTER TABLE    (allow wildcards in rules)

================================
Verifying schema...
================================
Table: clients
Table: endpoints
Table: notifications
Table: rules
(4 rows)

✓ Migration completed successfully!
```

## How to Run Again (If Needed)

The migration is idempotent (safe to run multiple times):

```bash
cd /path/to/alerting-platform
./scripts/deployment/run-migration.sh
```

The SQL uses `CREATE TABLE IF NOT EXISTS` and `DROP COLUMN IF EXISTS`, so it won't fail if tables already exist.

## Verification

To verify the schema was created correctly:

```bash
# View CloudWatch logs
aws logs tail /ecs/alerting-platform/prod/migration --region us-east-1

# Or connect to RDS and check tables
# (requires network access to private subnet)
psql -h alerting-platform-prod-postgres.cot8kqgoccg6.us-east-1.rds.amazonaws.com \
     -U postgres -d alerting -c "\dt"
```

## Next Steps

1. ✅ Database schema initialized
2. ⏳ **Create Kafka topics** (9 partitions each):
   - `alerts.new`
   - `rule.changed`
   - `alerts.matched`
   - `notifications.ready`
3. ⏳ Restart ECS services to connect to initialized database
4. ⏳ Verify end-to-end flow works

## Related Documentation

- **Prerequisites**: `docs/deployment/PREREQUISITES.md`
- **Deployment Guide**: `docs/deployment/PRODUCTION_DEPLOYMENT.md`
- **Terraform README**: `terraform/README.md`
- **Migration Guide**: `scripts/deployment/MIGRATION_GUIDE.md`
- **Active Context**: `memory-bank/activeContext.md`

## Troubleshooting

### Migration Failed

Check CloudWatch logs:
```bash
aws logs tail /ecs/alerting-platform/prod/migration --region us-east-1
```

Common issues:
- **DB password incorrect**: Check `terraform/terraform.tfvars`
- **Network connectivity**: Verify RDS security group allows ECS traffic
- **RDS not ready**: Wait a few minutes for RDS initialization

### Re-run Migration

Simply run the script again:
```bash
./scripts/deployment/run-migration.sh
```

The SQL is idempotent and safe to run multiple times.

## Cost Impact

- **ECR Storage**: ~5 MB for migration image (negligible)
- **CloudWatch Logs**: ~1 KB per migration run (negligible)
- **ECS Task**: Runs for <30 seconds (free tier)

**Total additional cost**: $0.00 (within free tier)

## Security Notes

1. **DB Password**: Stored in `terraform.tfvars` (git-ignored)
2. **Environment Variables**: Passed to ECS task at runtime
3. **Network**: RDS in private subnet, only accessible from ECS
4. **IAM**: Minimal permissions for migration task
5. **Logs**: CloudWatch logs contain no sensitive data

## Lessons Learned

1. **SSM Agent**: Amazon Linux 2 ECS-optimized AMI has SSM agent, but needs IAM policy to register
2. **Network Mode**: Bridge mode works well for EC2 launch type on t3.micro
3. **Docker Approach**: More reliable than SSM for one-off tasks
4. **Idempotency**: Always design migrations to be safely re-runnable
5. **Verification**: Include verification steps in migration script

## Conclusion

Database migration completed successfully using a production-ready, repeatable approach. The platform is now ready for Kafka topic creation and service deployment.
