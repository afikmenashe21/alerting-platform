#!/bin/bash
# Stop all infrastructure (Docker containers)
# This script stops Postgres, Kafka, Redis, Zookeeper, and Kafka UI

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

cd "$ROOT_DIR"

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

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}  Stopping Infrastructure${NC}"
echo -e "${YELLOW}========================================${NC}"
echo ""

# Check Docker
if ! command -v docker &> /dev/null; then
    echo_error "Docker is not installed or not in PATH"
    exit 1
fi

if ! docker info &> /dev/null; then
    echo_error "Docker is not running"
    exit 1
fi

# Check docker-compose
if command -v docker compose &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
elif command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE="docker-compose"
else
    echo_error "docker compose or docker-compose not found"
    exit 1
fi

# Stop infrastructure
echo_info "Stopping infrastructure containers..."
$DOCKER_COMPOSE down

echo ""
echo_success "Infrastructure stopped"
echo_info "To start again: make setup-infra"
