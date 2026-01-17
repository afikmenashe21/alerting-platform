#!/bin/bash
# Run aggregator service using centralized infrastructure
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

echo_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Step 1: Check Go
echo_step "1/4 Checking Go installation..."
if ! command -v go &> /dev/null; then
    echo_error "Go is not installed or not in PATH"
    exit 1
fi
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo_info "Go $GO_VERSION found ✓"
echo ""

# Step 2: Verify centralized infrastructure
echo_step "2/4 Verifying centralized infrastructure..."
if ! "$ROOT_DIR/scripts/infrastructure/verify-dependencies.sh"; then
    echo_error "Infrastructure dependencies are not available"
    echo_error "Start infrastructure with: cd $ROOT_DIR && make setup-infra"
    exit 1
fi
echo ""

# Step 3: Download Go dependencies
echo_step "3/4 Downloading Go dependencies..."
if ! go mod download 2>&1; then
    echo_error "Failed to download dependencies"
    exit 1
fi
echo_info "Dependencies downloaded ✓"
echo ""

# Step 4: Build and run
echo_step "4/4 Building and starting aggregator service..."
echo ""

# Build
if ! go build -o bin/aggregator ./cmd/aggregator; then
    echo_error "Failed to build aggregator"
    exit 1
fi

# Run with provided args or defaults
KAFKA_BROKERS="${KAFKA_BROKERS:-localhost:9092}"
POSTGRES_DSN="${POSTGRES_DSN:-postgres://postgres:postgres@127.0.0.1:5432/alerting?sslmode=disable}"

ARGS="${ARGS:--kafka-brokers $KAFKA_BROKERS -postgres-dsn $POSTGRES_DSN}"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Starting Aggregator Service${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo_info "Starting aggregator with: $ARGS"
echo ""
exec ./bin/aggregator $ARGS
