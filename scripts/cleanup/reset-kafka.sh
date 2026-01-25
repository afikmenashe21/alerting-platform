#!/bin/bash
# Reset Kafka consumer group offsets
# Run this from a container with kafka tools

KAFKA_BROKERS="${KAFKA_BROKERS:-34.201.202.8:9092}"

echo "Resetting Kafka consumer groups to latest..."
echo "Kafka: $KAFKA_BROKERS"

# Reset aggregator group
kafka-consumer-groups.sh --bootstrap-server $KAFKA_BROKERS --group aggregator-group --reset-offsets --to-latest --all-topics --execute 2>&1 || echo "aggregator-group reset failed or doesn't exist"

# Reset sender group  
kafka-consumer-groups.sh --bootstrap-server $KAFKA_BROKERS --group sender-group --reset-offsets --to-latest --all-topics --execute 2>&1 || echo "sender-group reset failed or doesn't exist"

# List current consumer groups
kafka-consumer-groups.sh --bootstrap-server $KAFKA_BROKERS --list 2>&1 || true

echo "Done"
