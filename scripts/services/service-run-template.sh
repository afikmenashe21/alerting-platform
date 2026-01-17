#!/bin/bash

# Template for service run scripts
# Services should verify dependencies but NOT manage infrastructure

set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ROOT_DIR="$(cd "$SERVICE_DIR/.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

echo_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

echo_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Step 1: Verify dependencies (DO NOT start them)
echo_info "Verifying dependencies..."
if ! "$ROOT_DIR/scripts/infrastructure/verify-dependencies.sh"; then
    echo_error "Dependencies are not available"
    echo_error "Start infrastructure with: cd $ROOT_DIR && make setup-infra"
    exit 1
fi

# Step 2: Build service
echo_info "Building service..."
cd "$SERVICE_DIR"
# Add your build command here
# go build -o bin/service ./cmd/service

# Step 3: Run service
echo_info "Starting service..."
# Add your run command here
# exec ./bin/service $ARGS
