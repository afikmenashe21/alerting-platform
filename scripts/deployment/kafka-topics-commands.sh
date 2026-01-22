#!/bin/bash

# Kafka Topics Creation Commands for AWS ECS
# This script provides the commands to run inside the Kafka container

echo "================================================"
echo "Kafka Topics Creation - Copy and Paste Commands"
echo "================================================"
echo ""
echo "Step 1: Get the Kafka task ARN and connect to it"
echo ""
echo "KAFKA_TASK=\$(aws ecs list-tasks --cluster alerting-platform-prod-cluster --service-name kafka --region us-east-1 --query 'taskArns[0]' --output text)"
echo "echo \"Kafka task: \$KAFKA_TASK\""
echo ""
echo "# Note: ECS Exec must be enabled on the task for this to work"
echo "# If this fails, you can create topics via a temporary ECS task (see alternative below)"
echo ""
echo "aws ecs execute-command \\"
echo "  --cluster alerting-platform-prod-cluster \\"
echo "  --task \$KAFKA_TASK \\"
echo "  --container kafka \\"
echo "  --interactive \\"
echo "  --command \"/bin/bash\" \\"
echo "  --region us-east-1"
echo ""
echo "================================================"
echo "Step 2: Once inside the Kafka container, run these commands:"
echo "================================================"
echo ""

# Define topics - note: rule.changed already exists with 3 partitions, we'll delete and recreate with 9
cat <<'EOF'
# First, check existing topics
kafka-topics --list --bootstrap-server localhost:9092

# Delete the auto-created rule.changed topic (only 3 partitions)
kafka-topics --delete --bootstrap-server localhost:9092 --topic rule.changed

# Create all topics with 9 partitions
kafka-topics --create --bootstrap-server localhost:9092 \
  --topic alerts.new \
  --partitions 9 \
  --replication-factor 1

kafka-topics --create --bootstrap-server localhost:9092 \
  --topic rule.changed \
  --partitions 9 \
  --replication-factor 1

kafka-topics --create --bootstrap-server localhost:9092 \
  --topic alerts.matched \
  --partitions 9 \
  --replication-factor 1

kafka-topics --create --bootstrap-server localhost:9092 \
  --topic notifications.ready \
  --partitions 9 \
  --replication-factor 1

# Verify all topics were created correctly
kafka-topics --list --bootstrap-server localhost:9092

# Describe topics to verify partition count
kafka-topics --describe --bootstrap-server localhost:9092

# Exit the container
exit

EOF

echo ""
echo "================================================"
echo "Alternative: Create topics using ECS RunTask"
echo "================================================"
echo ""
echo "If ECS Exec is not enabled, you can create a one-time task:"
echo ""
echo "See: scripts/deployment/create-kafka-topics-ecs-task.sh"
echo ""
