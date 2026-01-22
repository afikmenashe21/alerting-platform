#!/bin/bash
# Update all ECS services to use latest images

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

echo_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

echo_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Configuration
AWS_REGION="${AWS_REGION:-us-east-1}"
CLUSTER_NAME="${CLUSTER_NAME:-alerting-platform-prod-cluster}"

# Services to update
SERVICES=(
    "rule-service"
    "rule-updater"
    "evaluator"
    "aggregator"
    "sender"
    "alert-producer"
)

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Updating ECS Services${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo_info "Cluster: $CLUSTER_NAME"
echo_info "Region: $AWS_REGION"
echo ""

# Update each service
FAILED_SERVICES=()
for service in "${SERVICES[@]}"; do
    echo_info "Updating $service..."
    
    if aws ecs update-service \
        --cluster $CLUSTER_NAME \
        --service $service \
        --force-new-deployment \
        --region $AWS_REGION \
        --output text > /dev/null 2>&1; then
        echo_success "Updated $service"
    else
        echo_error "Failed to update $service"
        FAILED_SERVICES+=("$service")
    fi
done

echo ""
echo -e "${GREEN}========================================${NC}"

if [ ${#FAILED_SERVICES[@]} -eq 0 ]; then
    echo_success "All services updated successfully!"
    echo ""
    echo_info "Monitor deployments with:"
    echo "  aws ecs describe-services --cluster $CLUSTER_NAME \\"
    echo "    --services ${SERVICES[*]} --region $AWS_REGION"
    echo ""
    echo_info "View logs with:"
    echo "  aws logs tail /ecs/alerting-platform/prod/<service-name> --follow"
    exit 0
else
    echo_error "Failed to update ${#FAILED_SERVICES[@]} service(s): ${FAILED_SERVICES[*]}"
    exit 1
fi
