#!/bin/bash

# Fix Kafka-Zookeeper connection issue
# This script restarts Zookeeper and Kafka in the correct order

set -e

REGION="us-east-1"
CLUSTER="alerting-platform-prod-cluster"

echo "=================================="
echo "Fixing Kafka-Zookeeper Connection"
echo "=================================="
echo ""

# Step 1: Restart Zookeeper
echo "Step 1: Restarting Zookeeper service..."
aws ecs update-service \
  --cluster $CLUSTER \
  --service zookeeper \
  --force-new-deployment \
  --region $REGION \
  --query 'service.{Name:serviceName,Status:status,Running:runningCount,Desired:desiredCount}' \
  --output table

echo ""
echo "Waiting 45 seconds for Zookeeper to be fully ready..."
sleep 45

# Step 2: Check Zookeeper logs
echo ""
echo "Step 2: Checking Zookeeper logs..."
aws logs tail /ecs/alerting-platform/prod/zookeeper \
  --since 1m \
  --region $REGION \
  2>&1 | tail -20

# Step 3: Restart Kafka
echo ""
echo "Step 3: Restarting Kafka service..."
aws ecs update-service \
  --cluster $CLUSTER \
  --service kafka \
  --force-new-deployment \
  --region $REGION \
  --query 'service.{Name:serviceName,Status:status,Running:runningCount,Desired:desiredCount}' \
  --output table

echo ""
echo "Waiting 30 seconds for Kafka to start..."
sleep 30

# Step 4: Check Kafka logs
echo ""
echo "Step 4: Checking Kafka logs for Zookeeper connection..."
echo "Looking for successful Zookeeper connection..."
aws logs tail /ecs/alerting-platform/prod/kafka \
  --since 1m \
  --region $REGION \
  2>&1 | grep -i "zookeeper\|connection" | tail -10

# Step 5: Restart rule-service
echo ""
echo "Step 5: Restarting rule-service (scaling to desired count 1)..."
aws ecs update-service \
  --cluster $CLUSTER \
  --service rule-service \
  --desired-count 1 \
  --force-new-deployment \
  --region $REGION \
  --query 'service.{Name:serviceName,Status:status,Running:runningCount,Desired:desiredCount}' \
  --output table

echo ""
echo "Waiting 20 seconds for rule-service to stabilize..."
sleep 20

# Step 6: Check all services
echo ""
echo "Step 6: Checking all services status..."
aws ecs describe-services \
  --cluster $CLUSTER \
  --services kafka zookeeper rule-service evaluator aggregator sender rule-updater \
  --region $REGION \
  --query 'services[*].{Name:serviceName,Running:runningCount,Desired:desiredCount,Status:status,Deployment:deployments[0].rolloutState}' \
  --output table

echo ""
echo "=================================="
echo "Fix script completed!"
echo "=================================="
echo ""
echo "Next steps:"
echo "1. Monitor Kafka logs: aws logs tail /ecs/alerting-platform/prod/kafka --follow --region $REGION"
echo "2. Monitor rule-service logs: aws logs tail /ecs/alerting-platform/prod/rule-service --follow --region $REGION"
echo "3. Check for 'Successfully connected to Kafka' in rule-service logs"
echo "4. Once Kafka is healthy, run: ./scripts/deployment/create-kafka-topics.sh"
echo ""
