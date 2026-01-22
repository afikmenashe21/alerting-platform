#!/bin/bash

# Centralized Kafka topic creation
# All services share the same Kafka instance, so topics should be created centrally

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

KAFKA_CONTAINER="alerting-platform-kafka"

# Check if Kafka container is running
if ! docker ps --format "{{.Names}}" | grep -q "^${KAFKA_CONTAINER}$"; then
    echo_error "Kafka container '$KAFKA_CONTAINER' is not running"
    echo_error "Start it with: make setup-infra"
    exit 1
fi

# Find kafka-topics command
KAFKA_CMD=""
if docker exec "$KAFKA_CONTAINER" which kafka-topics &>/dev/null; then
    KAFKA_CMD="kafka-topics"
elif docker exec "$KAFKA_CONTAINER" ls /usr/bin/kafka-topics &>/dev/null; then
    KAFKA_CMD="/usr/bin/kafka-topics"
elif docker exec "$KAFKA_CONTAINER" ls /opt/kafka/bin/kafka-topics.sh &>/dev/null; then
    KAFKA_CMD="/opt/kafka/bin/kafka-topics.sh"
fi

if [ -z "$KAFKA_CMD" ]; then
    echo_error "Could not find kafka-topics command in container"
    exit 1
fi

echo_info "Creating Kafka topics..."

# Define all topics used by the platform
# Format: topic_name:partitions:replication_factor
# 9 partitions allows for better horizontal scaling (up to 9 consumer instances per service)
TOPICS=(
    "alerts.new:9:1"
    "rule.changed:9:1"
    "alerts.matched:9:1"
    "notifications.ready:9:1"
)

for topic_spec in "${TOPICS[@]}"; do
    IFS=':' read -r topic partitions replication <<< "$topic_spec"
    
    echo_info "Creating topic: $topic (partitions=$partitions, replication=$replication)"
    
    if docker exec "$KAFKA_CONTAINER" $KAFKA_CMD --create \
        --bootstrap-server localhost:9092 \
        --topic "$topic" \
        --partitions "$partitions" \
        --replication-factor "$replication" \
        --if-not-exists 2>/dev/null; then
        echo_success "Topic '$topic' created/verified"
    else
        echo_warn "Topic '$topic' may already exist or creation failed (continuing...)"
    fi
done

echo ""
echo_success "All Kafka topics created/verified"
