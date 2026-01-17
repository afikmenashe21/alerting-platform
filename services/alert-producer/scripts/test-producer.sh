#!/bin/bash
# Test script to verify alert-producer can connect to Kafka and produce events
# See docs/QUICKSTART.md for more information

set -e

echo "=== Testing Alert Producer ==="
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if Kafka is running
echo -e "${YELLOW}Checking if Kafka is running...${NC}"
if ! docker ps | grep -q kafka; then
    echo -e "${RED}Kafka is not running. Please start it with: docker-compose up -d${NC}"
    exit 1
fi
echo -e "${GREEN}Kafka is running${NC}"
echo ""

# Wait for Kafka to be ready
echo -e "${YELLOW}Waiting for Kafka to be ready...${NC}"
timeout=30
counter=0
until docker exec kafka kafka-broker-api-versions --bootstrap-server localhost:9092 > /dev/null 2>&1; do
    sleep 1
    counter=$((counter + 1))
    if [ $counter -ge $timeout ]; then
        echo -e "${RED}Kafka did not become ready within ${timeout} seconds${NC}"
        exit 1
    fi
done
echo -e "${GREEN}Kafka is ready${NC}"
echo ""

# Create topic if it doesn't exist
echo -e "${YELLOW}Creating topic 'alerts.new' if it doesn't exist...${NC}"
docker exec kafka kafka-topics --create \
    --bootstrap-server localhost:9092 \
    --topic alerts.new \
    --partitions 3 \
    --replication-factor 1 \
    --if-not-exists 2>/dev/null || echo "Topic already exists"
echo -e "${GREEN}Topic ready${NC}"
echo ""

# Build the producer
echo -e "${YELLOW}Building alert-producer...${NC}"
cd "$(dirname "$0")/.."
go build -o bin/alert-producer ./cmd/alert-producer
echo -e "${GREEN}Build complete${NC}"
echo ""

# Test 1: Burst mode - send 5 alerts
echo -e "${YELLOW}Test 1: Burst mode - sending 5 alerts...${NC}"
./bin/alert-producer \
    -burst 5 \
    -kafka-brokers localhost:9092 \
    -severity-dist "HIGH:50,MEDIUM:30,LOW:20" \
    -source-dist "api:100" \
    -name-dist "test:100" \
    2>&1 | grep -E "(Starting|Published first alert|Burst mode completed|error|ERROR)" || true
echo ""

# Verify messages were produced
echo -e "${YELLOW}Verifying messages in Kafka...${NC}"
MESSAGE_COUNT=$(docker exec kafka kafka-console-consumer \
    --bootstrap-server localhost:9092 \
    --topic alerts.new \
    --from-beginning \
    --max-messages 5 \
    --timeout-ms 5000 2>/dev/null | wc -l | tr -d ' ')

if [ "$MESSAGE_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✓ Successfully produced ${MESSAGE_COUNT} messages${NC}"
else
    echo -e "${RED}✗ No messages found in Kafka${NC}"
    exit 1
fi
echo ""

# Test 2: Verify message structure
echo -e "${YELLOW}Test 2: Verifying message structure...${NC}"
SAMPLE_MESSAGE=$(docker exec kafka kafka-console-consumer \
    --bootstrap-server localhost:9092 \
    --topic alerts.new \
    --from-beginning \
    --max-messages 1 \
    --timeout-ms 5000 2>/dev/null | head -1)

if [ -z "$SAMPLE_MESSAGE" ]; then
    echo -e "${RED}✗ Could not retrieve sample message${NC}"
    exit 1
fi

echo "Sample message:"
echo "$SAMPLE_MESSAGE" | jq '.' 2>/dev/null || echo "$SAMPLE_MESSAGE"
echo ""

# Check required fields
HAS_ALERT_ID=$(echo "$SAMPLE_MESSAGE" | jq -r '.alert_id // empty' 2>/dev/null)
HAS_SEVERITY=$(echo "$SAMPLE_MESSAGE" | jq -r '.severity // empty' 2>/dev/null)
HAS_SOURCE=$(echo "$SAMPLE_MESSAGE" | jq -r '.source // empty' 2>/dev/null)
HAS_NAME=$(echo "$SAMPLE_MESSAGE" | jq -r '.name // empty' 2>/dev/null)

if [ -n "$HAS_ALERT_ID" ] && [ -n "$HAS_SEVERITY" ] && [ -n "$HAS_SOURCE" ] && [ -n "$HAS_NAME" ]; then
    echo -e "${GREEN}✓ Message structure is valid${NC}"
    echo "  - alert_id: $HAS_ALERT_ID"
    echo "  - severity: $HAS_SEVERITY"
    echo "  - source: $HAS_SOURCE"
    echo "  - name: $HAS_NAME"
    
    # Verify severity is one of LOW, MEDIUM, HIGH
    if [[ "$HAS_SEVERITY" =~ ^(LOW|MEDIUM|HIGH)$ ]]; then
        echo -e "${GREEN}✓ Severity is valid enum (LOW/MEDIUM/HIGH)${NC}"
    else
        echo -e "${YELLOW}⚠ Severity '$HAS_SEVERITY' is not LOW/MEDIUM/HIGH${NC}"
    fi
else
    echo -e "${RED}✗ Message structure is invalid (missing required fields)${NC}"
    exit 1
fi
echo ""

# Test 3: Continuous mode for 2 seconds
echo -e "${YELLOW}Test 3: Continuous mode - 2 seconds at 5 RPS...${NC}"
timeout 3 ./bin/alert-producer \
    -rps 5 \
    -duration 2s \
    -kafka-brokers localhost:9092 \
    -severity-dist "HIGH:50,MEDIUM:30,LOW:20" \
    2>&1 | grep -E "(Starting|Published first alert|Progress update|Duration reached|error|ERROR)" || true
echo ""

echo -e "${GREEN}=== All tests completed successfully! ===${NC}"
echo ""
echo "You can view messages in Kafka UI at http://localhost:8080"
echo "Or consume messages with:"
echo "  docker exec kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic alerts.new --from-beginning"
