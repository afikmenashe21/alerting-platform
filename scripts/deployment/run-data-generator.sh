#!/bin/bash
# Run balanced test data generator via ECS task
# This cleans ALL data and generates fresh balanced test data

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
echo -e "${GREEN}  Balanced Test Data Generator via ECS${NC}"
echo -e "${GREEN}============================================${NC}"
echo ""

echo_warn "This will DELETE ALL DATA including notifications!"
read -p "Are you sure you want to continue? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo_info "Aborted"
    exit 0
fi

AWS_REGION="${AWS_REGION:-us-east-1}"
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
ECR_REPO="alerting-platform-prod-datagen"
IMAGE_TAG="latest"

# Get RDS connection info
cd terraform
RDS_ENDPOINT=$(terraform output -raw rds_endpoint 2>/dev/null)
RDS_HOST=$(echo "$RDS_ENDPOINT" | cut -d: -f1)
RDS_PORT=$(echo "$RDS_ENDPOINT" | cut -d: -f2)
DB_PASSWORD=$(grep '^db_password' terraform.tfvars | sed 's/.*"\(.*\)".*/\1/')
cd ..

echo_success "RDS Host: $RDS_HOST"
echo ""

# Step 1: Create ECR repository if it doesn't exist
echo_info "Creating ECR repository..."
aws ecr describe-repositories --repository-names "$ECR_REPO" --region "$AWS_REGION" >/dev/null 2>&1 || \
    aws ecr create-repository --repository-name "$ECR_REPO" --region "$AWS_REGION" >/dev/null

ECR_URI="${ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPO}"
echo_success "ECR Repository: $ECR_URI"
echo ""

# Step 2: Create temporary Dockerfile
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

cat > "$TMPDIR/Dockerfile" <<'DOCKERFILE'
FROM postgres:15-alpine
COPY generate-balanced-data.sql /scripts/generate-balanced-data.sql
COPY run.sh /run.sh
RUN chmod +x /run.sh
ENTRYPOINT ["/run.sh"]
DOCKERFILE

# Copy SQL file
cp scripts/test/test-data/generate-balanced-data.sql "$TMPDIR/"

# Create run script
cat > "$TMPDIR/run.sh" <<'RUNSCRIPT'
#!/bin/sh
set -e

echo "================================"
echo "Balanced Test Data Generator"
echo "================================"
echo ""

if [ -z "$DB_HOST" ] || [ -z "$DB_PASSWORD" ]; then
    echo "ERROR: DB_HOST and DB_PASSWORD must be set"
    exit 1
fi

DB_NAME="${DB_NAME:-alerting}"
DB_USER="${DB_USER:-postgres}"
DB_PORT="${DB_PORT:-5432}"

echo "Connecting to: $DB_HOST:$DB_PORT/$DB_NAME"
echo ""

# Wait for database to be ready
echo "Waiting for database to be ready..."
until PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c '\q' 2>/dev/null; do
    echo "  Database not ready, waiting..."
    sleep 2
done

echo "✓ Database is ready"
echo ""

# Run data generator
echo "Running balanced test data generator..."
echo "================================"
PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" < /scripts/generate-balanced-data.sql

echo ""
echo "✓ Data generation completed successfully!"
RUNSCRIPT

# Step 3: Build Docker image
echo_info "Building data generator Docker image..."
docker build --platform linux/amd64 -t "$ECR_REPO:$IMAGE_TAG" "$TMPDIR"
echo_success "Image built"
echo ""

# Step 4: Push to ECR
echo_info "Logging into ECR..."
aws ecr get-login-password --region "$AWS_REGION" | \
    docker login --username AWS --password-stdin "${ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"

echo_info "Tagging and pushing image..."
docker tag "$ECR_REPO:$IMAGE_TAG" "$ECR_URI:$IMAGE_TAG"
docker push "$ECR_URI:$IMAGE_TAG"
echo_success "Image pushed to ECR"
echo ""

# Step 5: Create CloudWatch log group
echo_info "Creating CloudWatch log group..."
aws logs create-log-group --log-group-name "/ecs/alerting-platform/prod/datagen" --region "$AWS_REGION" 2>/dev/null || true
echo_success "Log group ready"
echo ""

# Step 6: Register task definition
echo_info "Registering ECS task definition..."

TASK_DEF_FILE=$(mktemp)
trap "rm -f $TASK_DEF_FILE; rm -rf $TMPDIR" EXIT

cat > "$TASK_DEF_FILE" <<EOF
{
  "family": "alerting-platform-prod-datagen",
  "networkMode": "bridge",
  "requiresCompatibilities": ["EC2"],
  "cpu": "256",
  "memory": "256",
  "containerDefinitions": [
    {
      "name": "datagen",
      "image": "${ECR_URI}:${IMAGE_TAG}",
      "essential": true,
      "environment": [
        {"name": "DB_HOST", "value": "${RDS_HOST}"},
        {"name": "DB_PORT", "value": "${RDS_PORT}"},
        {"name": "DB_NAME", "value": "alerting"},
        {"name": "DB_USER", "value": "postgres"},
        {"name": "DB_PASSWORD", "value": "${DB_PASSWORD}"}
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/alerting-platform/prod/datagen",
          "awslogs-region": "${AWS_REGION}",
          "awslogs-stream-prefix": "datagen"
        }
      }
    }
  ]
}
EOF

TASK_DEF_ARN=$(aws ecs register-task-definition \
    --cli-input-json "file://$TASK_DEF_FILE" \
    --region "$AWS_REGION" \
    --query 'taskDefinition.taskDefinitionArn' \
    --output text)

echo_success "Task definition: $TASK_DEF_ARN"
echo ""

# Step 7: Run the task
echo_info "Running data generator task..."
TASK_ARN=$(aws ecs run-task \
    --cluster "alerting-platform-prod-cluster" \
    --task-definition "alerting-platform-prod-datagen" \
    --launch-type "EC2" \
    --region "$AWS_REGION" \
    --query 'tasks[0].taskArn' \
    --output text)

echo_success "Task started: $TASK_ARN"
echo ""

# Step 8: Wait for task to complete
echo_info "Waiting for data generation to complete..."
echo_warn "This may take 1-2 minutes..."
echo ""

aws ecs wait tasks-stopped \
    --cluster "alerting-platform-prod-cluster" \
    --tasks "$TASK_ARN" \
    --region "$AWS_REGION"

# Check task exit code
EXIT_CODE=$(aws ecs describe-tasks \
    --cluster "alerting-platform-prod-cluster" \
    --tasks "$TASK_ARN" \
    --region "$AWS_REGION" \
    --query 'tasks[0].containers[0].exitCode' \
    --output text)

echo ""
if [ "$EXIT_CODE" = "0" ]; then
    echo_success "Data generation completed successfully!"
    echo ""
    echo_info "Generated data:"
    echo "  - 200 clients"
    echo "  - 800 rules (4 per client)"
    echo "  - 1,600 endpoints (2 per rule)"
    echo "  - 0 notifications (cleaned)"
else
    echo_error "Data generation failed with exit code: $EXIT_CODE"
    echo_info "Check logs at:"
    echo "  aws logs tail /ecs/alerting-platform/prod/datagen --follow --region ${AWS_REGION}"
    exit 1
fi

echo ""
echo -e "${GREEN}============================================${NC}"
echo -e "${GREEN}  View logs:${NC}"
echo -e "${BLUE}  aws logs tail /ecs/alerting-platform/prod/datagen --follow --region ${AWS_REGION}${NC}"
echo -e "${GREEN}============================================${NC}"
