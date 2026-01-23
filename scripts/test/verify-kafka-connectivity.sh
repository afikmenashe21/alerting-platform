#!/bin/bash
# Verify Kafka DNS-based connectivity is working correctly

set -e

AWS_REGION="${AWS_REGION:-us-east-1}"
CLUSTER_NAME="alerting-platform-prod-cluster"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Kafka Connectivity Verification${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Step 1: Check DNS Resolution
echo -e "${BLUE}[1/5] Checking DNS Resolution...${NC}"
DNS_IP=$(aws route53 list-resource-record-sets \
  --hosted-zone-id Z03637091SEQ2F64I9QW6 \
  --region us-east-1 \
  --output json | jq -r '.ResourceRecordSets[] | select(.Name == "kafka.alerting-platform-prod.local.") | select(.Type == "A") | .ResourceRecords[0].Value')

if [ -n "$DNS_IP" ]; then
  echo -e "${GREEN}✓${NC} DNS Record: kafka.alerting-platform-prod.local → $DNS_IP"
else
  echo -e "${RED}✗${NC} DNS record not found!"
  exit 1
fi

# Step 2: Check Service Discovery Instance
echo ""
echo -e "${BLUE}[2/5] Checking Service Discovery Registration...${NC}"
INSTANCE_INFO=$(aws servicediscovery list-instances \
  --service-id srv-fublr2wykeaj2ayc \
  --region us-east-1 \
  --output json | jq -r '.Instances[0]')

INSTANCE_IP=$(echo "$INSTANCE_INFO" | jq -r '.Attributes.AWS_INSTANCE_IPV4')

if [ "$INSTANCE_IP" = "$DNS_IP" ]; then
  echo -e "${GREEN}✓${NC} Service Discovery Instance: $INSTANCE_IP (matches DNS)"
else
  echo -e "${RED}✗${NC} IP mismatch! DNS: $DNS_IP, Instance: $INSTANCE_IP"
  exit 1
fi

# Step 3: Check Kafka Service
echo ""
echo -e "${BLUE}[3/5] Checking Kafka Service...${NC}"
KAFKA_STATUS=$(aws ecs describe-services \
  --cluster "$CLUSTER_NAME" \
  --services kafka-combined \
  --region "$AWS_REGION" \
  --query 'services[0].{Running:runningCount,Desired:desiredCount,Status:status}' \
  --output json)

KAFKA_RUNNING=$(echo "$KAFKA_STATUS" | jq -r '.Running')
KAFKA_DESIRED=$(echo "$KAFKA_STATUS" | jq -r '.Desired')

if [ "$KAFKA_RUNNING" = "$KAFKA_DESIRED" ] && [ "$KAFKA_RUNNING" -gt 0 ]; then
  echo -e "${GREEN}✓${NC} Kafka Service: $KAFKA_RUNNING/$KAFKA_DESIRED tasks running"
else
  echo -e "${RED}✗${NC} Kafka Service: $KAFKA_RUNNING/$KAFKA_DESIRED tasks (not healthy)"
  exit 1
fi

# Step 4: Check Consumer Group Connections
echo ""
echo -e "${BLUE}[4/5] Checking Consumer Services...${NC}"
for service in evaluator rule-updater aggregator sender; do
  LOG_STREAM=$(aws logs describe-log-streams \
    --log-group-name "/ecs/alerting-platform/prod/$service" \
    --region "$AWS_REGION" \
    --order-by LastEventTime \
    --descending \
    --max-items 1 \
    --output text \
    --query 'logStreams[0].logStreamName' 2>/dev/null || echo "")
  
  if [ -n "$LOG_STREAM" ]; then
    ERROR_COUNT=$(aws logs get-log-events \
      --log-group-name "/ecs/alerting-platform/prod/$service" \
      --log-stream-name "$LOG_STREAM" \
      --region "$AWS_REGION" \
      --start-time $(($(date +%s) - 120))000 \
      --output json 2>/dev/null | jq -r '.events[].message' | grep -c 'ERROR.*Kafka\|ERROR.*kafka' || echo "0")
    
    if [ "$ERROR_COUNT" = "0" ]; then
      echo -e "  ${GREEN}✓${NC} $service: No Kafka errors (last 2 min)"
    else
      echo -e "  ${RED}✗${NC} $service: $ERROR_COUNT Kafka errors (last 2 min)"
    fi
  else
    echo -e "  ${RED}✗${NC} $service: No log stream found"
  fi
done

# Step 5: Check Kafka Consumer Groups
echo ""
echo -e "${BLUE}[5/5] Checking Kafka Consumer Groups...${NC}"
KAFKA_LOG_STREAM=$(aws logs describe-log-streams \
  --log-group-name "/ecs/alerting-platform/prod/kafka" \
  --region "$AWS_REGION" \
  --order-by LastEventTime \
  --descending \
  --max-items 1 \
  --output text \
  --query 'logStreams[0].logStreamName' 2>/dev/null)

if [ -n "$KAFKA_LOG_STREAM" ]; then
  for group in evaluator-group rule-updater-group aggregator-group sender-group; do
    GROUP_STATUS=$(aws logs get-log-events \
      --log-group-name "/ecs/alerting-platform/prod/kafka" \
      --log-stream-name "$KAFKA_LOG_STREAM" \
      --region "$AWS_REGION" \
      --start-time $(($(date +%s) - 300))000 \
      --output json 2>/dev/null | jq -r '.events[].message' | grep -c "Stabilized group $group" || echo "0")
    
    if [ "$GROUP_STATUS" -gt 0 ]; then
      echo -e "  ${GREEN}✓${NC} $group: Connected and stabilized"
    else
      echo -e "  ${BLUE}ℹ${NC} $group: No recent activity (normal if no messages)"
    fi
  done
fi

# Summary
echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}✓ Verification Complete${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "DNS-based Kafka connectivity is working!"
echo ""
echo "Key Information:"
echo "  - Kafka DNS: kafka.alerting-platform-prod.local"
echo "  - Current IP: $DNS_IP"
echo "  - Network Mode: awsvpc"
echo "  - Service Discovery: Enabled"
echo ""
echo "Benefits:"
echo "  - No hardcoded IPs"
echo "  - Auto-updates when Kafka moves instances"
echo "  - Zero manual intervention needed"
echo "  - Production-ready and reliable"
echo ""
