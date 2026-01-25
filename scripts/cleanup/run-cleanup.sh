#!/bin/bash
# Full system cleanup - clears Redis, resets Kafka offsets, truncates notifications

set -e

cd "$(dirname "$0")"

GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${GREEN}========================================"
echo -e "  Building and Running System Cleanup"
echo -e "========================================${NC}"
echo ""

AWS_REGION="${AWS_REGION:-us-east-1}"
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
ECR_REPO="alerting-platform-prod-cleanup"

# Get infrastructure info from terraform
cd ../../terraform
EC2_IP=$(terraform output -raw api_elastic_ip 2>/dev/null)
KAFKA_HOST="$EC2_IP"  # Kafka runs on EC2
REDIS_HOST=$(terraform output -raw redis_endpoint 2>/dev/null | cut -d: -f1)
RDS_HOST=$(terraform output -json rds_endpoint 2>/dev/null | tr -d '"' | cut -d: -f1)
DB_PASSWORD=$(grep '^db_password' terraform.tfvars | sed 's/.*"\(.*\)".*/\1/')
cd ../scripts/cleanup

echo -e "${BLUE}[INFO]${NC} Kafka: $KAFKA_HOST:9092"
echo -e "${BLUE}[INFO]${NC} Redis: $REDIS_HOST:6379"
echo -e "${BLUE}[INFO]${NC} RDS: $RDS_HOST"
echo ""

# Create ECR repo if needed
aws ecr describe-repositories --repository-names "$ECR_REPO" --region "$AWS_REGION" >/dev/null 2>&1 || \
    aws ecr create-repository --repository-name "$ECR_REPO" --region "$AWS_REGION" >/dev/null

ECR_URI="${ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPO}"

# Build and push
echo -e "${BLUE}[INFO]${NC} Building cleanup image..."
docker build --platform linux/amd64 -t "$ECR_REPO:latest" .

echo -e "${BLUE}[INFO]${NC} Pushing to ECR..."
aws ecr get-login-password --region "$AWS_REGION" | docker login --username AWS --password-stdin "${ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"
docker tag "$ECR_REPO:latest" "$ECR_URI:latest"
docker push "$ECR_URI:latest"

# Create log group
aws logs create-log-group --log-group-name "/ecs/alerting-platform/prod/cleanup" --region "$AWS_REGION" 2>/dev/null || true

# Register task definition
TASK_DEF=$(cat <<EOF
{
  "family": "alerting-platform-prod-cleanup",
  "networkMode": "bridge",
  "requiresCompatibilities": ["EC2"],
  "cpu": "256",
  "memory": "512",
  "containerDefinitions": [
    {
      "name": "cleanup",
      "image": "${ECR_URI}:latest",
      "essential": true,
      "environment": [
        {"name": "KAFKA_BROKERS", "value": "${KAFKA_HOST}:9092"},
        {"name": "REDIS_HOST", "value": "${REDIS_HOST}"},
        {"name": "REDIS_PORT", "value": "6379"},
        {"name": "DB_HOST", "value": "${RDS_HOST}"},
        {"name": "DB_PORT", "value": "5432"},
        {"name": "DB_NAME", "value": "alerting"},
        {"name": "DB_USER", "value": "postgres"},
        {"name": "DB_PASSWORD", "value": "${DB_PASSWORD}"}
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
)

echo "$TASK_DEF" > /tmp/cleanup-task.json
aws ecs register-task-definition --cli-input-json file:///tmp/cleanup-task.json --region "$AWS_REGION" >/dev/null

# Run task
echo -e "${BLUE}[INFO]${NC} Running cleanup task..."
TASK_ARN=$(aws ecs run-task \
    --cluster "alerting-platform-prod-cluster" \
    --task-definition "alerting-platform-prod-cleanup" \
    --launch-type "EC2" \
    --region "$AWS_REGION" \
    --query 'tasks[0].taskArn' \
    --output text)

echo -e "${BLUE}[INFO]${NC} Task: $TASK_ARN"
echo -e "${BLUE}[INFO]${NC} Waiting for completion..."

aws ecs wait tasks-stopped --cluster "alerting-platform-prod-cluster" --tasks "$TASK_ARN" --region "$AWS_REGION"

EXIT_CODE=$(aws ecs describe-tasks --cluster "alerting-platform-prod-cluster" --tasks "$TASK_ARN" --region "$AWS_REGION" \
    --query 'tasks[0].containers[0].exitCode' --output text)

echo ""
if [ "$EXIT_CODE" = "0" ]; then
    echo -e "${GREEN}✓ Cleanup completed successfully!${NC}"
else
    echo "✗ Cleanup failed with exit code: $EXIT_CODE"
    echo "Check logs: aws logs tail /ecs/alerting-platform/prod/cleanup --follow --region $AWS_REGION"
    exit 1
fi

echo ""
echo -e "${BLUE}[INFO]${NC} View logs: aws logs tail /ecs/alerting-platform/prod/cleanup --region $AWS_REGION"
