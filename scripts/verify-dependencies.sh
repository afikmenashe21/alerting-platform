#!/bin/bash

# Centralized dependency verification script
# This script verifies that all shared infrastructure is running and accessible
# Services should call this before starting, but NOT manage the infrastructure themselves

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

# Check if a port is accessible
check_port() {
    local port=$1
    local service=$2
    
    if command -v nc &> /dev/null; then
        if nc -z 127.0.0.1 "$port" 2>/dev/null; then
            return 0
        fi
    elif command -v timeout &> /dev/null; then
        if timeout 1 bash -c "echo > /dev/tcp/127.0.0.1/$port" 2>/dev/null; then
            return 0
        fi
    fi
    return 1
}

# Check if a container is running
check_container() {
    local container_name=$1
    if docker ps --format "{{.Names}}" | grep -q "^${container_name}$"; then
        return 0
    fi
    return 1
}

# Verify Postgres
verify_postgres() {
    echo_info "Verifying Postgres..."
    
    local container_name="alerting-platform-postgres"
    if ! check_container "$container_name"; then
        echo_error "Postgres container '$container_name' is not running"
        echo_error "Start it with: docker compose up -d postgres"
        return 1
    fi
    
    if ! check_port 5432; then
        echo_error "Postgres port 5432 is not accessible"
        return 1
    fi
    
    # Test connection
    if ! docker exec "$container_name" pg_isready -U postgres &> /dev/null 2>&1; then
        echo_error "Postgres is not ready to accept connections"
        return 1
    fi
    
    # Verify database exists
    if ! docker exec "$container_name" psql -U postgres -lqt | cut -d \| -f 1 | grep -qw alerting; then
        echo_warn "Database 'alerting' does not exist, creating it..."
        docker exec "$container_name" psql -U postgres -c "CREATE DATABASE alerting;" 2>/dev/null || {
            echo_error "Failed to create database 'alerting'"
            return 1
        }
    fi
    
    echo_success "Postgres is running and accessible"
    return 0
}

# Verify Zookeeper
verify_zookeeper() {
    echo_info "Verifying Zookeeper..."
    
    local container_name="alerting-platform-zookeeper"
    if ! check_container "$container_name"; then
        echo_error "Zookeeper container '$container_name' is not running"
        echo_error "Start it with: docker compose up -d zookeeper"
        return 1
    fi
    
    if ! check_port 2181; then
        echo_error "Zookeeper port 2181 is not accessible"
        return 1
    fi
    
    echo_success "Zookeeper is running and accessible"
    return 0
}

# Verify Kafka
verify_kafka() {
    echo_info "Verifying Kafka..."
    
    local container_name="alerting-platform-kafka"
    if ! check_container "$container_name"; then
        echo_error "Kafka container '$container_name' is not running"
        echo_error "Start it with: docker compose up -d kafka"
        return 1
    fi
    
    if ! check_port 9092; then
        echo_error "Kafka port 9092 is not accessible"
        return 1
    fi
    
    # Test Kafka connection
    if ! docker exec "$container_name" kafka-broker-api-versions --bootstrap-server localhost:9092 &> /dev/null 2>&1; then
        echo_error "Kafka is not ready to accept connections"
        return 1
    fi
    
    echo_success "Kafka is running and accessible"
    return 0
}

# Verify Redis
verify_redis() {
    echo_info "Verifying Redis..."
    
    local container_name="alerting-platform-redis"
    if ! check_container "$container_name"; then
        echo_error "Redis container '$container_name' is not running"
        echo_error "Start it with: docker compose up -d redis"
        return 1
    fi
    
    if ! check_port 6379; then
        echo_error "Redis port 6379 is not accessible"
        return 1
    fi
    
    # Test Redis connection
    if ! docker exec "$container_name" redis-cli ping &> /dev/null 2>&1; then
        echo_error "Redis is not ready to accept connections"
        return 1
    fi
    
    echo_success "Redis is running and accessible"
    return 0
}

# Verify MailHog
verify_mailhog() {
    echo_info "Verifying MailHog..."
    
    local container_name="alerting-platform-mailhog"
    if ! check_container "$container_name"; then
        echo_error "MailHog container '$container_name' is not running"
        echo_error "Start it with: docker compose up -d mailhog"
        return 1
    fi
    
    if ! check_port 1025; then
        echo_error "MailHog SMTP port 1025 is not accessible"
        return 1
    fi
    
    if ! check_port 8025; then
        echo_warn "MailHog web UI port 8025 is not accessible (optional)"
    fi
    
    echo_success "MailHog is running and accessible"
    echo_info "  SMTP: localhost:1025"
    echo_info "  Web UI: http://localhost:8025"
    return 0
}

# Main verification
main() {
    local all_ok=true
    
    verify_postgres || all_ok=false
    verify_zookeeper || all_ok=false
    verify_kafka || all_ok=false
    verify_redis || all_ok=false
    verify_mailhog || all_ok=false
    
    if [ "$all_ok" = true ]; then
        echo ""
        echo_success "All dependencies are running and accessible"
        return 0
    else
        echo ""
        echo_error "Some dependencies are not available"
        echo_error "Start all infrastructure with: docker compose up -d"
        return 1
    fi
}

# Run main function
main "$@"
