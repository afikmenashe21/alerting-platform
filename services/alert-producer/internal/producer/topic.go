// Package producer provides a Kafka producer wrapper for publishing alerts.
package producer

import (
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
)

// createTopicIfNotExists attempts to create the topic if it doesn't exist.
// This is a best-effort operation and failures are logged but don't prevent producer creation.
func createTopicIfNotExists(broker, topic string) {
	conn, err := kafka.Dial("tcp", broker)
	if err != nil {
		slog.Warn("Could not connect to Kafka to check/create topic",
			"broker", broker,
			"topic", topic,
			"error", err,
			"note", "Topic may need to be created manually",
		)
		return
	}
	defer conn.Close()

	// Check if topic exists
	partitions, err := conn.ReadPartitions(topic)
	if err == nil && len(partitions) > 0 {
		slog.Info("Topic already exists",
			"topic", topic,
			"partitions", len(partitions),
		)
		return
	}

	// Topic doesn't exist, try to create it
	topicConfig := kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:    3,
		ReplicationFactor: 1,
	}

	err = conn.CreateTopics(topicConfig)
	if err != nil {
		slog.Warn("Could not create topic (may need to be created manually)",
			"topic", topic,
			"error", err,
			"tip", "Run: docker exec kafka kafka-topics --create --bootstrap-server localhost:9092 --topic "+topic+" --partitions 3 --replication-factor 1",
		)
		return
	}

	slog.Info("Created topic",
		"topic", topic,
		"partitions", 3,
		"replication_factor", 1,
	)

	// Wait for topic to be fully available (Kafka topic creation is asynchronous)
	// Retry reading partitions up to 5 times with 1 second delay
	maxRetries := 5
	retryDelay := 1 * time.Second
	for i := 0; i < maxRetries; i++ {
		time.Sleep(retryDelay)
		partitions, err := conn.ReadPartitions(topic)
		if err == nil && len(partitions) > 0 {
			slog.Info("Topic is now available",
				"topic", topic,
				"partitions", len(partitions),
			)
			return
		}
		if i < maxRetries-1 {
			slog.Info("Waiting for topic to be available",
				"topic", topic,
				"attempt", i+1,
				"max_retries", maxRetries,
			)
		}
	}

	// If we still can't read partitions, try one more time with a fresh connection
	// This handles the case where the connection was stale
	verifyConn, err := kafka.Dial("tcp", broker)
	if err == nil {
		defer verifyConn.Close()
		partitions, err := verifyConn.ReadPartitions(topic)
		if err == nil && len(partitions) > 0 {
			slog.Info("Topic is now available (verified with new connection)",
				"topic", topic,
				"partitions", len(partitions),
			)
			return
		}
	}

	slog.Warn("Topic created but may not be fully available yet",
		"topic", topic,
		"note", "Producer will retry on first write if topic is not ready",
	)
}
