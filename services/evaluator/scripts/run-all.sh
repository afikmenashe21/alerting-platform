#!/bin/bash
# Run evaluator service using centralized infrastructure
# This script verifies dependencies and runs the service - it does NOT manage infrastructure

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ROOT_DIR="$(cd "$SERVICE_DIR/../.." && pwd)"

cd "$SERVICE_DIR"

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

echo_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Step 1: Check Go
echo_step "1/5 Checking Go installation..."
if ! command -v go &> /dev/null; then
    echo_error "Go is not installed or not in PATH"
    exit 1
fi
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo_info "Go $GO_VERSION found ✓"
echo ""

# Step 2: Verify centralized infrastructure
echo_step "2/5 Verifying centralized infrastructure..."
if ! "$ROOT_DIR/scripts/verify-dependencies.sh"; then
    echo_error "Infrastructure dependencies are not available"
    echo_error "Start infrastructure with: cd $ROOT_DIR && make setup-infra"
    exit 1
fi
echo ""

# Step 3: Download Go dependencies
echo_step "3/5 Downloading Go dependencies..."
if ! go mod download 2>&1; then
    echo_error "Failed to download dependencies"
    exit 1
fi
echo_info "Dependencies downloaded ✓"
echo ""

# Step 4: Check if rule snapshot exists (optional - rule-updater should create it)
echo_step "4/5 Checking for rule snapshot in Redis..."
REDIS_CONTAINER="alerting-platform-redis"
if docker ps --format "{{.Names}}" | grep -q "^${REDIS_CONTAINER}$"; then
    SNAPSHOT_EXISTS=$(docker exec "$REDIS_CONTAINER" redis-cli EXISTS rules:snapshot 2>/dev/null | grep -q "1" && echo "yes" || echo "no")
    if [ "$SNAPSHOT_EXISTS" != "yes" ]; then
        echo_warn "Rule snapshot not found in Redis"
        echo_warn "This is OK - rule-updater will create it when rules are added"
        echo_warn "Or create a test snapshot with: cd scripts && go run create-test-snapshot.go localhost:6379"
    else
        echo_info "Rule snapshot exists in Redis ✓"
    fi
else
    echo_warn "Could not check Redis snapshot (container not found)"
fi
echo ""

# Step 5: Build and run
echo_step "5/5 Building and starting evaluator service..."
echo ""

# Build
if ! go build -o bin/evaluator ./cmd/evaluator; then
    echo_error "Failed to build evaluator"
    exit 1
fi

# Run with provided args or defaults
KAFKA_BROKERS="${KAFKA_BROKERS:-localhost:9092}"
REDIS_ADDR="${REDIS_ADDR:-localhost:6379}"
VERSION_POLL_INTERVAL="${VERSION_POLL_INTERVAL:-5s}"

ARGS="${ARGS:--kafka-brokers $KAFKA_BROKERS -redis-addr $REDIS_ADDR -version-poll-interval $VERSION_POLL_INTERVAL}"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Starting Evaluator Service${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo_info "Starting evaluator with: $ARGS"
echo ""
exec ./bin/evaluator $ARGS
