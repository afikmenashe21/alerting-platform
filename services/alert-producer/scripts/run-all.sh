#!/bin/bash
# Run alert-producer service using centralized infrastructure
# This script verifies dependencies and runs the service - it does NOT manage infrastructure
#
# Infrastructure should be started centrally from the root directory:
#   cd ../.. && make setup-infra
#
# This script:
# - Verifies infrastructure is running
# - Downloads Go dependencies
# - Builds the service
# - Runs the service

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

# Step 2: Verify centralized infrastructure (only Kafka needed for alert-producer)
echo_step "2/5 Verifying centralized infrastructure..."
if ! "$ROOT_DIR/scripts/infrastructure/verify-dependencies.sh"; then
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

# Step 4: Build the service
echo_step "4/5 Building alert-producer..."
if ! go build -o bin/alert-producer ./cmd/alert-producer 2>&1; then
    echo_error "Failed to build alert-producer"
    exit 1
fi
echo_info "Build successful ✓"
echo ""

# Step 5: Run the service
echo_step "5/5 Starting alert-producer..."
echo ""

# Build command arguments with defaults
KAFKA_BROKERS="${KAFKA_BROKERS:-localhost:9092}"
TOPIC="${TOPIC:-alerts.new}"
RPS="${RPS:-10}"
DURATION="${DURATION:-60s}"
BURST="${BURST:-0}"

ARGS="${ARGS:--kafka-brokers $KAFKA_BROKERS -topic $TOPIC}"

if [ "$BURST" != "0" ]; then
    ARGS="${ARGS} -burst $BURST"
else
    ARGS="${ARGS} -rps $RPS -duration $DURATION"
fi

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Starting Alert Producer${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo_info "Starting alert-producer with: $ARGS"
echo_info "Publishing alerts to topic: $TOPIC"
echo ""
echo -e "${YELLOW}Press Ctrl+C to stop the service${NC}"
echo ""

exec ./bin/alert-producer $ARGS
