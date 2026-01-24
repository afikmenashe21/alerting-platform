#!/bin/bash
# Build and push all Docker images to ECR

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

echo_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Check if AWS CLI is installed
if ! command -v aws &> /dev/null; then
    echo_error "AWS CLI not found. Please install it first."
    exit 1
fi

# Check if Docker is running
if ! docker info &> /dev/null; then
    echo_error "Docker is not running. Please start Docker."
    exit 1
fi

# Get AWS account ID
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
if [ -z "$AWS_ACCOUNT_ID" ]; then
    echo_error "Failed to get AWS account ID. Check your AWS credentials."
    exit 1
fi

# Get AWS region
AWS_REGION="${AWS_REGION:-us-east-1}"
echo_info "Using AWS region: $AWS_REGION"

# ECR base URL
ECR_BASE="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/alerting-platform"

# Services to build
SERVICES=(
    "rule-service"
    "rule-updater"
    "evaluator"
    "aggregator"
    "sender"
    "alert-producer"
    "metrics-service"
)

# Get image tag (default to latest, or use git commit hash)
IMAGE_TAG="${IMAGE_TAG:-latest}"
if command -v git &> /dev/null && [ -d .git ]; then
    GIT_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo "")
    if [ -n "$GIT_HASH" ]; then
        IMAGE_TAG="$GIT_HASH"
        echo_info "Using git commit hash as tag: $IMAGE_TAG"
    fi
fi

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Building and Pushing Docker Images${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo_info "AWS Account: $AWS_ACCOUNT_ID"
echo_info "Region: $AWS_REGION"
echo_info "Image Tag: $IMAGE_TAG"
echo_info "Services: ${SERVICES[*]}"
echo ""

# Authenticate Docker to ECR
echo_info "Authenticating Docker to ECR..."
aws ecr get-login-password --region $AWS_REGION | \
    docker login --username AWS --password-stdin ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com

if [ $? -ne 0 ]; then
    echo_error "Failed to authenticate to ECR"
    exit 1
fi
echo_success "Authenticated to ECR"
echo ""

# Build and push each service
FAILED_SERVICES=()
for service in "${SERVICES[@]}"; do
    echo_info "========================================"
    echo_info "Building $service..."
    echo_info "========================================"
    
    # Full image name
    IMAGE_NAME="${ECR_BASE}/${service}:${IMAGE_TAG}"
    IMAGE_NAME_LATEST="${ECR_BASE}/${service}:latest"
    
    # Build image (from project root!)
    # Use --platform linux/amd64 to ensure compatibility with ECS EC2 instances
    if docker build \
        --platform linux/amd64 \
        -f services/$service/Dockerfile \
        -t $IMAGE_NAME \
        -t $IMAGE_NAME_LATEST \
        .; then
        echo_success "Built $service"
    else
        echo_error "Failed to build $service"
        FAILED_SERVICES+=("$service")
        continue
    fi
    
    # Push image with tag
    echo_info "Pushing $IMAGE_NAME..."
    if docker push $IMAGE_NAME; then
        echo_success "Pushed $IMAGE_NAME"
    else
        echo_error "Failed to push $IMAGE_NAME"
        FAILED_SERVICES+=("$service")
        continue
    fi
    
    # Push image with 'latest' tag
    if [ "$IMAGE_TAG" != "latest" ]; then
        echo_info "Pushing $IMAGE_NAME_LATEST..."
        if docker push $IMAGE_NAME_LATEST; then
            echo_success "Pushed $IMAGE_NAME_LATEST"
        else
            echo_warn "Failed to push $IMAGE_NAME_LATEST (non-critical)"
        fi
    fi
    
    echo_success "Completed $service"
    echo ""
done

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Build and Push Summary${NC}"
echo -e "${GREEN}========================================${NC}"

if [ ${#FAILED_SERVICES[@]} -eq 0 ]; then
    echo_success "All services built and pushed successfully!"
    echo ""
    echo_info "Next steps:"
    echo "  1. Update ECS services to use new images:"
    echo "     ./scripts/deployment/update-services.sh"
    echo ""
    echo "  2. Or manually update individual services:"
    echo "     aws ecs update-service --cluster alerting-platform-prod-cluster \\"
    echo "       --service <service-name> --force-new-deployment"
    echo ""
    exit 0
else
    echo_error "Failed to build/push ${#FAILED_SERVICES[@]} service(s):"
    for service in "${FAILED_SERVICES[@]}"; do
        echo "  - $service"
    done
    exit 1
fi
