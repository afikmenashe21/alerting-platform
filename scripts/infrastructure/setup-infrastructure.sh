#!/bin/bash

# Centralized infrastructure setup script
# This script starts all shared infrastructure (Postgres, Kafka, Zookeeper, Redis, MailHog)
# Services should call verify-dependencies.sh, NOT this script

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

echo_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

echo_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Get script directory (root of alerting-platform)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

cd "$ROOT_DIR"

echo_info "Setting up shared infrastructure..."
echo_info "This will start: Postgres, Zookeeper, Kafka, Redis, MailHog"

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

# Start infrastructure
echo_info "Starting infrastructure services..."
$DOCKER_COMPOSE up -d postgres zookeeper kafka redis mailhog

# Wait for services to be ready
echo_info "Waiting for services to be ready..."

# Wait for Postgres
echo_info "Waiting for Postgres..."
for i in {1..30}; do
    if docker exec alerting-platform-postgres pg_isready -U postgres &> /dev/null 2>&1; then
        echo_success "Postgres is ready"
        break
    fi
    if [ $i -eq 30 ]; then
        echo_error "Postgres did not become ready in time"
        exit 1
    fi
    sleep 1
done

# Wait for Zookeeper
echo_info "Waiting for Zookeeper..."
for i in {1..20}; do
    if docker exec alerting-platform-zookeeper nc -z localhost 2181 &> /dev/null 2>&1; then
        echo_success "Zookeeper is ready"
        break
    fi
    if [ $i -eq 20 ]; then
        echo_warn "Zookeeper may not be fully ready"
    fi
    sleep 1
done

# Wait for Kafka
echo_info "Waiting for Kafka..."
# First, ensure container is running
for i in {1..30}; do
    if docker ps --format "{{.Names}}" | grep -q "^alerting-platform-kafka$"; then
        break
    fi
    if [ $i -eq 30 ]; then
        echo_error "Kafka container failed to start"
        exit 1
    fi
    sleep 1
done

# Now wait for Kafka to be ready
KAFKA_READY=false
for i in {1..60}; do
    # Check if container is still running
    if ! docker ps --format "{{.Names}}" | grep -q "^alerting-platform-kafka$"; then
        echo_error "Kafka container stopped unexpectedly"
        docker logs --tail 20 alerting-platform-kafka
        exit 1
    fi
    
    # Try multiple methods to check Kafka readiness
    if docker exec alerting-platform-kafka kafka-broker-api-versions --bootstrap-server localhost:9092 &> /dev/null 2>&1; then
        KAFKA_READY=true
        echo_success "Kafka is ready"
        break
    fi
    
    # Show progress every 5 iterations
    if [ $((i % 5)) -eq 0 ]; then
        echo_info "  Still waiting for Kafka... (${i}/60)"
    fi
    
    if [ $i -eq 60 ]; then
        echo_warn "Kafka may not be fully ready after 2 minutes"
        echo_info "Checking Kafka logs..."
        docker logs --tail 30 alerting-platform-kafka
    fi
    sleep 2
done

if [ "$KAFKA_READY" = false ]; then
    echo_warn "Proceeding anyway, but Kafka may not be fully ready"
fi

# Wait for Redis
echo_info "Waiting for Redis..."
for i in {1..20}; do
    if docker exec alerting-platform-redis redis-cli ping &> /dev/null 2>&1; then
        echo_success "Redis is ready"
        break
    fi
    if [ $i -eq 20 ]; then
        echo_warn "Redis may not be fully ready"
    fi
    sleep 1
done

# Wait for MailHog
echo_info "Waiting for MailHog..."
for i in {1..20}; do
    if docker exec alerting-platform-mailhog wget --quiet --tries=1 --spider http://localhost:8025 &> /dev/null 2>&1; then
        echo_success "MailHog is ready"
        break
    fi
    if [ $i -eq 20 ]; then
        echo_warn "MailHog may not be fully ready"
    fi
    sleep 1
done

echo ""
echo_success "All infrastructure services are running"
echo_info "Run './scripts/infrastructure/verify-dependencies.sh' to verify connectivity"
echo_info ""
echo_info "MailHog Web UI: http://localhost:8025 (view captured emails)"
echo_info "MailHog SMTP: localhost:1025 (for sending emails)"
