#!/bin/bash
# Comprehensive cleanup script - clears Redis metrics and database notifications
# Run this via ECS to access both Redis and RDS

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
echo_success() { echo -e "${GREEN}[✓]${NC} $1"; }
echo_error() { echo -e "${RED}[ERROR]${NC} $1"; }
echo_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }

cd "$(dirname "$0")/../.."

echo -e "${GREEN}============================================${NC}"
echo -e "${GREEN}  Comprehensive Cleanup (Redis + DB)${NC}"
echo -e "${GREEN}============================================${NC}"
echo ""

echo_warn "This will:"
echo_warn "  1. Delete ALL Redis metrics keys (resets service counters)"
echo_warn "  2. DELETE all notifications from database"
echo_warn "  3. Restart all services to reset in-memory counters"
echo ""
read -p "Are you sure you want to continue? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo_info "Aborted"
    exit 0
fi

AWS_REGION="${AWS_REGION:-us-east-1}"
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
ECR_REPO="alerting-platform-prod-cleanup"
IMAGE_TAG="latest"

# Get connection info from terraform
cd terraform
RDS_ENDPOINT=$(terraform output -raw rds_endpoint 2>/dev/null)
RDS_HOST=$(echo "$RDS_ENDPOINT" | cut -d: -f1)
RDS_PORT=$(echo "$RDS_ENDPOINT" | cut -d: -f2)
DB_PASSWORD=$(grep '^db_password' terraform.tfvars | sed 's/.*"\(.*\)".*/\1/')
REDIS_ENDPOINT=$(terraform output -raw redis_endpoint 2>/dev/null || echo "")
cd ..

echo_success "RDS Host: $RDS_HOST"
echo_success "Redis: $REDIS_ENDPOINT"
echo ""

# Create temporary build directory
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

# Create Dockerfile
cat > "$TMPDIR/Dockerfile" <<'DOCKERFILE'
FROM postgres:15-alpine

# Install redis-cli
RUN apk add --no-cache redis

COPY cleanup.sh /cleanup.sh
RUN chmod +x /cleanup.sh

ENTRYPOINT ["/cleanup.sh"]
DOCKERFILE

# Create cleanup script
cat > "$TMPDIR/cleanup.sh" <<'SCRIPT'
#!/bin/sh
set -e

echo "============================================"
echo "  Comprehensive Cleanup"
echo "============================================"
echo ""

DB_NAME="${DB_NAME:-alerting}"
DB_USER="${DB_USER:-postgres}"
DB_PORT="${DB_PORT:-5432}"

# Wait for database
echo "Connecting to database..."
until PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c '\q' 2>/dev/null; do
    echo "  Waiting for database..."
    sleep 2
done
echo "✓ Database connected"

# Step 1: Clear Redis metrics
if [ -n "$REDIS_HOST" ]; then
    echo ""
    echo "Clearing Redis metrics keys..."

    # Delete all metrics:* keys
    DELETED=$(redis-cli -h "$REDIS_HOST" -p "${REDIS_PORT:-6379}" --scan --pattern "metrics:*" | xargs -r redis-cli -h "$REDIS_HOST" -p "${REDIS_PORT:-6379}" DEL 2>/dev/null || echo "0")
    echo "✓ Deleted metrics keys: $DELETED"

    # Optionally clear rules snapshot to force rebuild
    # redis-cli -h "$REDIS_HOST" -p "${REDIS_PORT:-6379}" DEL rules:snapshot rules:version
else
    echo "⚠ Redis not configured, skipping Redis cleanup"
fi

# Step 2: Delete all notifications
echo ""
echo "Deleting all notifications..."
DELETED_COUNT=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "DELETE FROM notifications RETURNING 1;" | wc -l | tr -d ' ')
echo "✓ Deleted $DELETED_COUNT notifications"

# Step 3: Update pg_stat statistics
echo ""
echo "Updating statistics..."
PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "ANALYZE notifications;"
echo "✓ Statistics updated"

# Step 4: Verify counts
echo ""
echo "============================================"
echo "  Verification"
echo "============================================"
PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "
SELECT 'clients' as table_name, COUNT(*) as count FROM clients
UNION ALL SELECT 'rules', COUNT(*) FROM rules
UNION ALL SELECT 'endpoints', COUNT(*) FROM endpoints
UNION ALL SELECT 'notifications', COUNT(*) FROM notifications;
"

echo ""
echo "✓ Cleanup completed!"
SCRIPT

# Step 1: Create ECR repo
echo_info "Creating ECR repository..."
aws ecr describe-repositories --repository-names "$ECR_REPO" --region "$AWS_REGION" >/dev/null 2>&1 || \
    aws ecr create-repository --repository-name "$ECR_REPO" --region "$AWS_REGION" >/dev/null

ECR_URI="${ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPO}"

# Step 2: Build and push
echo_info "Building cleanup image..."
docker build --platform linux/amd64 -t "$ECR_REPO:$IMAGE_TAG" "$TMPDIR" >/dev/null 2>&1
echo_success "Image built"

echo_info "Pushing to ECR..."
aws ecr get-login-password --region "$AWS_REGION" | docker login --username AWS --password-stdin "${ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com" >/dev/null 2>&1
docker tag "$ECR_REPO:$IMAGE_TAG" "$ECR_URI:$IMAGE_TAG"
docker push "$ECR_URI:$IMAGE_TAG" >/dev/null 2>&1
echo_success "Image pushed"

# Step 3: Create log group
aws logs create-log-group --log-group-name "/ecs/alerting-platform/prod/cleanup" --region "$AWS_REGION" 2>/dev/null || true

# Step 4: Get Redis host (parse from endpoint)
REDIS_HOST=$(echo "$REDIS_ENDPOINT" | cut -d: -f1)
REDIS_PORT=$(echo "$REDIS_ENDPOINT" | cut -d: -f2)
[ -z "$REDIS_PORT" ] && REDIS_PORT="6379"

# Step 5: Register task definition
echo_info "Registering task definition..."
TASK_DEF_FILE=$(mktemp)

cat > "$TASK_DEF_FILE" <<EOF
{
  "family": "alerting-platform-prod-cleanup",
  "networkMode": "bridge",
  "requiresCompatibilities": ["EC2"],
  "cpu": "128",
  "memory": "256",
  "containerDefinitions": [
    {
      "name": "cleanup",
      "image": "${ECR_URI}:${IMAGE_TAG}",
      "essential": true,
      "environment": [
        {"name": "DB_HOST", "value": "${RDS_HOST}"},
        {"name": "DB_PORT", "value": "${RDS_PORT}"},
        {"name": "DB_NAME", "value": "alerting"},
        {"name": "DB_USER", "value": "postgres"},
        {"name": "DB_PASSWORD", "value": "${DB_PASSWORD}"},
        {"name": "REDIS_HOST", "value": "${REDIS_HOST}"},
        {"name": "REDIS_PORT", "value": "${REDIS_PORT}"}
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/alerting-platform/prod/cleanup",
          "awslogs-region": "${AWS_REGION}",
          "awslogs-stream-prefix": "cleanup"
        }
      }
    }
  ]
}
EOF

aws ecs register-task-definition --cli-input-json "file://$TASK_DEF_FILE" --region "$AWS_REGION" >/dev/null 2>&1
rm "$TASK_DEF_FILE"
echo_success "Task definition registered"

# Step 6: Run cleanup task
echo_info "Running cleanup task..."
TASK_ARN=$(aws ecs run-task \
    --cluster "alerting-platform-prod-cluster" \
    --task-definition "alerting-platform-prod-cleanup" \
    --launch-type "EC2" \
    --region "$AWS_REGION" \
    --query 'tasks[0].taskArn' \
    --output text 2>&1)

if [[ "$TASK_ARN" == "None" || -z "$TASK_ARN" ]]; then
    echo_error "Failed to start cleanup task"
    aws ecs run-task --cluster "alerting-platform-prod-cluster" --task-definition "alerting-platform-prod-cleanup" --launch-type "EC2" --region "$AWS_REGION" --query 'failures' --output json
    exit 1
fi

echo_success "Task started: $TASK_ARN"

# Step 7: Wait for completion
echo_info "Waiting for cleanup to complete..."
aws ecs wait tasks-stopped --cluster "alerting-platform-prod-cluster" --tasks "$TASK_ARN" --region "$AWS_REGION"

EXIT_CODE=$(aws ecs describe-tasks --cluster "alerting-platform-prod-cluster" --tasks "$TASK_ARN" --region "$AWS_REGION" --query 'tasks[0].containers[0].exitCode' --output text)

if [ "$EXIT_CODE" = "0" ]; then
    echo_success "Cleanup completed successfully!"
else
    echo_error "Cleanup failed with exit code: $EXIT_CODE"
    echo_info "Check logs: aws logs tail /ecs/alerting-platform/prod/cleanup --follow --region $AWS_REGION"
    exit 1
fi

# Step 8: Restart services to reset in-memory counters
echo ""
echo_info "Restarting services to reset in-memory counters..."

SERVICES="rule-service rule-updater evaluator aggregator sender alert-producer metrics-service"
for svc in $SERVICES; do
    aws ecs update-service --cluster alerting-platform-prod-cluster --service "$svc" --force-new-deployment --region "$AWS_REGION" --output text >/dev/null 2>&1
    echo_success "Restarted $svc"
done

echo ""
echo -e "${GREEN}============================================${NC}"
echo -e "${GREEN}  Cleanup Complete!${NC}"
echo -e "${GREEN}============================================${NC}"
echo ""
echo_info "All services are restarting with fresh counters"
echo_info "Redis metrics will be refreshed within 30 seconds"
echo_info "Database notifications have been deleted"
