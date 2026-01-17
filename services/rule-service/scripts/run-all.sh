#!/bin/bash
# Run rule-service using centralized infrastructure
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

echo_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
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

# Step 2: Verify centralized infrastructure
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
echo_step "4/5 Building rule-service..."
if ! go build -o bin/rule-service ./cmd/rule-service 2>&1; then
    echo_error "Failed to build rule-service"
    exit 1
fi
echo_info "Build successful ✓"
echo ""

# Step 5: Run the service
echo_step "5/5 Starting rule-service..."
echo ""

# Build command arguments with defaults
HTTP_PORT="${HTTP_PORT:-8081}"
KAFKA_BROKERS="${KAFKA_BROKERS:-localhost:9092}"
RULE_CHANGED_TOPIC="${RULE_CHANGED_TOPIC:-rule.changed}"
POSTGRES_DSN="${POSTGRES_DSN:-postgres://postgres:postgres@127.0.0.1:5432/alerting?sslmode=disable}"

# Check if HTTP port is available
echo_info "Checking if port $HTTP_PORT is available..."
PORT_IN_USE=false
if command -v nc &> /dev/null; then
    if nc -z localhost "$HTTP_PORT" 2>/dev/null; then
        PORT_IN_USE=true
    fi
elif command -v lsof &> /dev/null; then
    if lsof -Pi :$HTTP_PORT -sTCP:LISTEN -t >/dev/null 2>&1; then
        PORT_IN_USE=true
    fi
fi

if [ "$PORT_IN_USE" = "true" ]; then
    echo_warn "Port $HTTP_PORT is already in use!"
    echo_error "Cannot start rule-service - port $HTTP_PORT is in use"
    exit 1
fi

echo_info "Port $HTTP_PORT is available ✓"
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Starting Rule Service${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

ARGS="${ARGS:--http-port $HTTP_PORT -kafka-brokers $KAFKA_BROKERS -rule-changed-topic $RULE_CHANGED_TOPIC -postgres-dsn $POSTGRES_DSN}"

echo_info "Starting rule-service with: $ARGS"
echo ""
echo_info "Service will be available at: http://localhost:$HTTP_PORT"
echo_info "API endpoints:"
echo_info "  - POST   /api/v1/clients"
echo_info "  - GET    /api/v1/clients?client_id=<id>"
echo_info "  - GET    /api/v1/clients"
echo_info "  - POST   /api/v1/rules"
echo_info "  - GET    /api/v1/rules?rule_id=<id>"
echo_info "  - GET    /api/v1/rules?client_id=<id>"
echo_info "  - GET    /api/v1/rules"
echo_info "  - PUT    /api/v1/rules/update?rule_id=<id>"
echo_info "  - POST   /api/v1/rules/toggle?rule_id=<id>"
echo_info "  - DELETE /api/v1/rules/delete?rule_id=<id>"
echo_info ""
echo_info "Endpoint endpoints:"
echo_info "  - POST   /api/v1/endpoints"
echo_info "  - GET    /api/v1/endpoints?endpoint_id=<id>"
echo_info "  - GET    /api/v1/endpoints?rule_id=<id>"
echo_info "  - PUT    /api/v1/endpoints/update?endpoint_id=<id>"
echo_info "  - POST   /api/v1/endpoints/toggle?endpoint_id=<id>"
echo_info "  - DELETE /api/v1/endpoints/delete?endpoint_id=<id>"
echo_info ""
echo_info "Notification endpoints:"
echo_info "  - GET    /api/v1/notifications?notification_id=<id>"
echo_info "  - GET    /api/v1/notifications?client_id=<id>&status=<status>"
echo_info ""
echo_info "  - GET    /health"
echo ""
echo -e "${YELLOW}Press Ctrl+C to stop the service${NC}"
echo ""

exec ./bin/rule-service $ARGS
