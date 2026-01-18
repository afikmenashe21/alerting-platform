#!/bin/bash
# Send a single test alert (LOW/test-source/test-name) via alert-producer
# This is a quick test to verify the alerting pipeline

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
SERVICE_DIR="$ROOT_DIR/services/alert-producer"

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

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Sending Single Test Alert${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo_info "Alert details:"
echo_info "  Severity: LOW"
echo_info "  Source: test-source"
echo_info "  Name: test-name"
echo ""

# Step 1: Check Go
echo_info "Checking Go installation..."
if ! command -v go &> /dev/null; then
    echo_error "Go is not installed or not in PATH"
    exit 1
fi
echo_success "Go found"
echo ""

# Step 2: Verify infrastructure (Kafka)
echo_info "Verifying Kafka is running..."
if ! "$ROOT_DIR/scripts/infrastructure/verify-dependencies.sh" > /dev/null 2>&1; then
    echo_error "Infrastructure is not running"
    echo_error "Start infrastructure with: make setup-infra"
    exit 1
fi
echo_success "Kafka is running"
echo ""

# Step 3: Download dependencies
echo_info "Downloading Go dependencies..."
if ! go mod download 2>&1; then
    echo_error "Failed to download dependencies"
    exit 1
fi
echo_success "Dependencies downloaded"
echo ""

# Step 4: Build
echo_info "Building alert-producer..."
if ! go build -o bin/alert-producer ./cmd/alert-producer 2>&1; then
    echo_error "Failed to build alert-producer"
    exit 1
fi
echo_success "Build successful"
echo ""

# Step 5: Run single test
echo_info "Sending single test alert..."
echo ""

KAFKA_BROKERS="${KAFKA_BROKERS:-localhost:9092}"
TOPIC="${TOPIC:-alerts.new}"

if ./bin/alert-producer -single-test -kafka-brokers "$KAFKA_BROKERS" -topic "$TOPIC"; then
    echo ""
    echo_success "Test alert sent successfully!"
    echo_info "Check your services to see the alert flow through the pipeline"
else
    echo ""
    echo_error "Failed to send test alert"
    exit 1
fi
