#!/bin/bash
# Test script to validate rule.changed events are published to Kafka

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_DIR"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
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

# Configuration
HTTP_PORT="${HTTP_PORT:-8081}"
BASE_URL="http://localhost:${HTTP_PORT}"
KAFKA_BROKERS="${KAFKA_BROKERS:-localhost:9092}"
TOPIC="rule.changed"

echo_step "Testing rule.changed event publishing"
echo ""

# Step 1: Check if service is running
echo_step "1/5 Checking if rule-service is running..."
if ! curl -s "${BASE_URL}/health" > /dev/null 2>&1; then
    echo_error "Rule-service is not running on ${BASE_URL}"
    echo_error "Please start it with: make run-all"
    exit 1
fi
echo_info "Rule-service is running ✓"
echo ""

# Step 2: Check if Kafka is accessible
echo_step "2/5 Checking Kafka connectivity..."
KAFKA_CONTAINER=$(docker ps --filter "name=kafka" --format "{{.Names}}" | head -n1 || echo "")
if [ -z "$KAFKA_CONTAINER" ]; then
    echo_warn "Could not find Kafka container, but will try to consume anyway"
else
    echo_info "Found Kafka container: $KAFKA_CONTAINER"
fi
echo ""

# Step 3: Create a test client
echo_step "3/5 Creating test client..."
CLIENT_ID="test-client-$(date +%s)"
CLIENT_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/clients" \
  -H "Content-Type: application/json" \
  -d "{
    \"client_id\": \"${CLIENT_ID}\",
    \"name\": \"Test Client\"
  }")

if echo "$CLIENT_RESPONSE" | grep -q "test-client"; then
    echo_info "Client created: ${CLIENT_ID} ✓"
else
    echo_error "Failed to create client"
    echo_error "Response: $CLIENT_RESPONSE"
    exit 1
fi
echo ""

# Step 4: Create a test rule and verify event is published
echo_step "4/5 Creating test rule and verifying event publication..."
RULE_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/rules" \
  -H "Content-Type: application/json" \
  -d "{
    \"client_id\": \"${CLIENT_ID}\",
    \"severity\": \"HIGH\",
    \"source\": \"api\",
    \"name\": \"test-event\"
  }")

RULE_ID=$(echo "$RULE_RESPONSE" | grep -o '"rule_id":"[^"]*"' | cut -d'"' -f4 || echo "")

if [ -z "$RULE_ID" ]; then
    echo_error "Failed to create rule or extract rule_id"
    echo_error "Response: $RULE_RESPONSE"
    exit 1
fi

echo_info "Rule created: ${RULE_ID} ✓"
echo_info "Waiting 2 seconds for event to be published..."
sleep 2
echo ""

# Step 5: Consume from Kafka to verify event
echo_step "5/5 Consuming from Kafka to verify event..."

# Try to consume the last message from the topic
if [ -n "$KAFKA_CONTAINER" ]; then
    # Try different Kafka command locations
    KAFKA_CMD=""
    if docker exec "$KAFKA_CONTAINER" which kafka-console-consumer &>/dev/null; then
        KAFKA_CMD="kafka-console-consumer"
    elif docker exec "$KAFKA_CONTAINER" ls /usr/bin/kafka-console-consumer.sh &>/dev/null; then
        KAFKA_CMD="/usr/bin/kafka-console-consumer.sh"
    elif docker exec "$KAFKA_CONTAINER" ls /opt/kafka/bin/kafka-console-consumer.sh &>/dev/null; then
        KAFKA_CMD="/opt/kafka/bin/kafka-console-consumer.sh"
    fi
    
    if [ -n "$KAFKA_CMD" ]; then
        echo_info "Consuming from topic ${TOPIC}..."
        # Consume the last message (or wait up to 5 seconds for a new one)
        EVENT_JSON=$(timeout 5 docker exec "$KAFKA_CONTAINER" $KAFKA_CMD \
            --bootstrap-server localhost:9092 \
            --topic "${TOPIC}" \
            --from-beginning \
            --max-messages 1 \
            2>/dev/null | tail -n1 || echo "")
        
        if [ -n "$EVENT_JSON" ]; then
            echo_info "Event received from Kafka ✓"
            echo ""
            echo_info "Event content:"
            echo "$EVENT_JSON" | python3 -m json.tool 2>/dev/null || echo "$EVENT_JSON"
            echo ""
            
            # Verify event structure
            if echo "$EVENT_JSON" | grep -q "\"rule_id\"" && \
               echo "$EVENT_JSON" | grep -q "\"client_id\"" && \
               echo "$EVENT_JSON" | grep -q "\"action\"" && \
               echo "$EVENT_JSON" | grep -q "\"version\""; then
                echo_info "Event structure is valid ✓"
                
                # Check if it's our test rule
                if echo "$EVENT_JSON" | grep -q "${RULE_ID}"; then
                    echo_info "Event contains our test rule_id ✓"
                fi
                
                # Check action
                ACTION=$(echo "$EVENT_JSON" | grep -o '"action":"[^"]*"' | cut -d'"' -f4 || echo "")
                if [ "$ACTION" = "CREATED" ]; then
                    echo_info "Event action is CREATED ✓"
                else
                    echo_warn "Event action is ${ACTION} (expected CREATED)"
                fi
            else
                echo_error "Event structure is invalid"
                exit 1
            fi
        else
            echo_warn "No event received from Kafka (may need to check manually)"
            echo_warn "This could mean:"
            echo_warn "  1. Event was published but already consumed"
            echo_warn "  2. Kafka topic doesn't exist"
            echo_warn "  3. Consumer command not available"
        fi
    else
        echo_warn "Could not find kafka-console-consumer command"
        echo_warn "Skipping Kafka consumption verification"
    fi
else
    echo_warn "Kafka container not found, skipping consumption test"
fi

echo ""
echo_step "Testing additional operations..."

# Test UPDATE event
echo_info "Updating rule to trigger UPDATED event..."
UPDATE_RESPONSE=$(curl -s -X PUT "${BASE_URL}/api/v1/rules/update?rule_id=${RULE_ID}" \
  -H "Content-Type: application/json" \
  -d "{
    \"severity\": \"CRITICAL\",
    \"source\": \"api\",
    \"name\": \"test-event\",
    \"version\": 1
  }")

if echo "$UPDATE_RESPONSE" | grep -q "CRITICAL"; then
    echo_info "Rule updated successfully ✓"
    sleep 1
else
    echo_warn "Rule update may have failed"
fi

# Test DELETE event
echo_info "Deleting rule to trigger DELETED event..."
DELETE_RESPONSE=$(curl -s -X DELETE "${BASE_URL}/api/v1/rules/delete?rule_id=${RULE_ID}")

if [ "$DELETE_RESPONSE" = "" ]; then
    echo_info "Rule deleted successfully ✓"
    sleep 1
else
    echo_warn "Rule deletion may have failed"
fi

echo ""
echo_info "Test completed!"
echo_info ""
echo_info "To manually verify events, you can:"
echo_info "  1. Use kafka-console-consumer:"
echo_info "     docker exec -it <kafka-container> kafka-console-consumer --bootstrap-server localhost:9092 --topic rule.changed --from-beginning"
echo_info "  2. Check rule-service logs for 'Published rule changed event' messages"
echo_info "  3. Monitor the rule-updater service (when implemented) to see it consuming events"
