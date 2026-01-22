#!/bin/bash
# Build migration Docker image, push to ECR, and run as ECS task

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
echo -e "${GREEN}  Database Migration via ECS${NC}"
echo -e "${GREEN}============================================${NC}"
echo ""

AWS_REGION="${AWS_REGION:-us-east-1}"
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
ECR_REPO="alerting-platform-prod-migration"
IMAGE_TAG="latest"

# Get RDS connection info
cd terraform
RDS_ENDPOINT=$(terraform output -raw rds_endpoint 2>/dev/null)
RDS_HOST=$(echo "$RDS_ENDPOINT" | cut -d: -f1)
RDS_PORT=$(echo "$RDS_ENDPOINT" | cut -d: -f2)
DB_PASSWORD=$(grep '^db_password' terraform.tfvars | sed 's/.*"\(.*\)".*/\1/')
cd ..

echo_success "RDS Host: $RDS_HOST"
echo_success "RDS Port: $RDS_PORT"
echo ""

# Step 1: Create ECR repository if it doesn't exist
echo_info "Creating ECR repository..."
aws ecr describe-repositories --repository-names "$ECR_REPO" --region "$AWS_REGION" >/dev/null 2>&1 || \
    aws ecr create-repository --repository-name "$ECR_REPO" --region "$AWS_REGION" >/dev/null

ECR_URI="${ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPO}"
echo_success "ECR Repository: $ECR_URI"
echo ""

# Step 2: Build Docker image
echo_info "Building migration Docker image..."
cd migrations
docker build --platform linux/amd64 -t "$ECR_REPO:$IMAGE_TAG" .
cd ..
echo_success "Image built"
echo ""

# Step 3: Push to ECR
echo_info "Logging into ECR..."
aws ecr get-login-password --region "$AWS_REGION" | \
    docker login --username AWS --password-stdin "${ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"

echo_info "Tagging and pushing image..."
docker tag "$ECR_REPO:$IMAGE_TAG" "$ECR_URI:$IMAGE_TAG"
docker push "$ECR_URI:$IMAGE_TAG"
echo_success "Image pushed to ECR"
echo ""

# Step 4: Create CloudWatch log group
echo_info "Creating CloudWatch log group..."
aws logs create-log-group --log-group-name "/ecs/alerting-platform/prod/migration" --region "$AWS_REGION" 2>/dev/null || true
echo_success "Log group ready"
echo ""

# Step 5: Register task definition
echo_info "Registering ECS task definition..."

# Create temporary file for task definition
TASK_DEF_FILE=$(mktemp)
trap "rm -f $TASK_DEF_FILE" EXIT

cat > "$TASK_DEF_FILE" <<EOF
{
  "family": "alerting-platform-prod-migration",
  "networkMode": "bridge",
  "requiresCompatibilities": ["EC2"],
  "cpu": "256",
  "memory": "512",
  "containerDefinitions": [
    {
      "name": "migration",
      "image": "${ECR_URI}:${IMAGE_TAG}",
      "essential": true,
      "environment": [
        {
          "name": "DB_HOST",
          "value": "${RDS_HOST}"
        },
        {
          "name": "DB_PORT",
          "value": "${RDS_PORT}"
        },
        {
          "name": "DB_NAME",
          "value": "alerting"
        },
        {
          "name": "DB_USER",
          "value": "postgres"
        },
        {
          "name": "DB_PASSWORD",
          "value": "${DB_PASSWORD}"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/alerting-platform/prod/migration",
          "awslogs-region": "${AWS_REGION}",
          "awslogs-stream-prefix": "migration"
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

# Step 6: Run the task
echo_info "Running migration task..."
TASK_ARN=$(aws ecs run-task \
    --cluster "alerting-platform-prod-cluster" \
    --task-definition "alerting-platform-prod-migration" \
    --launch-type "EC2" \
    --region "$AWS_REGION" \
    --query 'tasks[0].taskArn' \
    --output text)

echo_success "Task started: $TASK_ARN"
echo ""

# Step 7: Wait for task to complete
echo_info "Waiting for migration to complete..."
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
    echo_success "Migration completed successfully!"
else
    echo_error "Migration failed with exit code: $EXIT_CODE"
    echo_info "Check logs at: https://console.aws.amazon.com/cloudwatch/home?region=${AWS_REGION}#logsV2:log-groups/log-group//ecs/alerting-platform/prod/migration"
    exit 1
fi

echo ""
echo -e "${GREEN}============================================${NC}"
echo -e "${GREEN}  View logs:${NC}"
echo -e "${BLUE}  aws logs tail /ecs/alerting-platform/prod/migration --follow --region ${AWS_REGION}${NC}"
echo -e "${GREEN}============================================${NC}"
