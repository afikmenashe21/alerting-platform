#!/bin/bash

# Increase Kafka Topic Partitions
# Increases partition count from 1 to 9 for all alerting topics

set -e

CLUSTER="alerting-platform-prod-cluster"
REGION="us-east-1"
KAFKA_BROKER="10.0.1.109:9092"

echo "================================================"
echo "Increasing Kafka Topic Partitions 1 -> 9"
echo "================================================"
echo ""
echo "Cluster: $CLUSTER"
echo "Region: $REGION"
echo "Kafka Broker: $KAFKA_BROKER"
echo ""

# Get Kafka image
echo "Getting Kafka container image..."
KAFKA_IMAGE=$(aws ecs describe-task-definition \
  --task-definition alerting-platform-prod-kafka \
  --region $REGION \
  --query 'taskDefinition.containerDefinitions[0].image' \
  --output text)
echo "Using image: $KAFKA_IMAGE"
echo ""

# Create the topic alteration commands
TOPIC_COMMANDS="
echo '==== Current topic configuration ====';
kafka-topics --describe --bootstrap-server $KAFKA_BROKER --exclude-internal;

echo '';
echo '==== Increasing partitions to 9 ====';

kafka-topics --alter --bootstrap-server $KAFKA_BROKER \
  --topic alerts.new \
  --partitions 9 || echo 'Failed to alter alerts.new';

kafka-topics --alter --bootstrap-server $KAFKA_BROKER \
  --topic rule.changed \
  --partitions 9 || echo 'Failed to alter rule.changed';

kafka-topics --alter --bootstrap-server $KAFKA_BROKER \
  --topic alerts.matched \
  --partitions 9 || echo 'Failed to alter alerts.matched';

kafka-topics --alter --bootstrap-server $KAFKA_BROKER \
  --topic notifications.ready \
  --partitions 9 || echo 'Failed to alter notifications.ready';

echo '';
echo '==== Verifying updated configuration ====';
kafka-topics --describe --bootstrap-server $KAFKA_BROKER --exclude-internal;

echo '';
echo '✅ Done! All topics now have 9 partitions.';
"

# Create task definition
cat > /tmp/kafka-partition-task.json <<EOF
{
  "family": "kafka-partition-updater",
  "networkMode": "host",
  "requiresCompatibilities": ["EC2"],
  "cpu": "256",
  "memory": "512",
  "containerDefinitions": [
    {
      "name": "kafka-partitions",
      "image": "$KAFKA_IMAGE",
      "essential": true,
      "command": [
        "/bin/bash",
        "-c",
        $(echo "$TOPIC_COMMANDS" | jq -Rs .)
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/alerting-platform/prod/kafka-topics",
          "awslogs-region": "$REGION",
          "awslogs-stream-prefix": "partition-updater"
        }
      }
    }
  ]
}
EOF

# Register task definition
echo "Registering task definition..."
TASK_DEF_ARN=$(aws ecs register-task-definition \
  --cli-input-json file:///tmp/kafka-partition-task.json \
  --region $REGION \
  --query 'taskDefinition.taskDefinitionArn' \
  --output text)

echo "Task definition: $TASK_DEF_ARN"
echo ""

# Run the task
echo "Running task to increase partitions..."
TASK_ARN=$(aws ecs run-task \
  --cluster $CLUSTER \
  --task-definition $TASK_DEF_ARN \
  --launch-type EC2 \
  --region $REGION \
  --query 'tasks[0].taskArn' \
  --output text)

echo "Task started: $TASK_ARN"
echo ""

# Wait for task
echo "Waiting for task to complete..."
aws ecs wait tasks-stopped \
  --cluster $CLUSTER \
  --tasks $TASK_ARN \
  --region $REGION

# Get status
TASK_STATUS=$(aws ecs describe-tasks \
  --cluster $CLUSTER \
  --tasks $TASK_ARN \
  --region $REGION \
  --query 'tasks[0].containers[0].exitCode' \
  --output text)

echo ""
echo "Task completed with exit code: $TASK_STATUS"
echo ""

# Show logs
echo "================================================"
echo "Task Output:"
echo "================================================"
echo ""
aws logs tail /ecs/alerting-platform/prod/kafka-topics --since 2m --region $REGION | grep -A 50 "partition-updater"

# Cleanup
rm -f /tmp/kafka-partition-task.json

echo ""
echo "================================================"
echo "Done!"
echo "================================================"
echo ""

if [ "$TASK_STATUS" = "0" ]; then
  echo "✅ Partitions increased to 9 successfully!"
  exit 0
else
  echo "❌ Task failed. Check logs above."
  exit 1
fi
