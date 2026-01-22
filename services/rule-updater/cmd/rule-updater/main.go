package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"rule-updater/internal/config"
	"rule-updater/internal/consumer"
	"rule-updater/internal/database"
	"rule-updater/internal/processor"
	"rule-updater/internal/snapshot"

	"github.com/redis/go-redis/v9"
)

func main() {
	// Parse command-line flags with environment variable fallbacks
	cfg := &config.Config{}
	flag.StringVar(&cfg.KafkaBrokers, "kafka-brokers", getEnvOrDefault("KAFKA_BROKERS", "localhost:9092"), "Kafka broker addresses (comma-separated)")
	flag.StringVar(&cfg.RuleChangedTopic, "rule-changed-topic", getEnvOrDefault("RULE_CHANGED_TOPIC", "rule.changed"), "Kafka topic for rule change events")
	flag.StringVar(&cfg.ConsumerGroupID, "consumer-group-id", getEnvOrDefault("CONSUMER_GROUP_ID", "rule-updater-group"), "Kafka consumer group ID")
	flag.StringVar(&cfg.PostgresDSN, "postgres-dsn", getEnvOrDefault("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable"), "PostgreSQL connection string")
	flag.StringVar(&cfg.RedisAddr, "redis-addr", getEnvOrDefault("REDIS_ADDR", "localhost:6379"), "Redis server address")
	flag.Parse()

	// Set up structured logging
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	slog.Info("Starting rule-updater service",
		"kafka_brokers", cfg.KafkaBrokers,
		"rule_changed_topic", cfg.RuleChangedTopic,
		"consumer_group_id", cfg.ConsumerGroupID,
		"postgres_dsn", maskDSN(cfg.PostgresDSN),
		"redis_addr", cfg.RedisAddr,
	)

	if err := cfg.Validate(); err != nil {
		slog.Error("Invalid configuration", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		slog.Info("Received shutdown signal, shutting down gracefully...")
		cancel()
	}()

	// Initialize database connection
	slog.Info("Connecting to PostgreSQL database")
	db, err := database.NewDB(cfg.PostgresDSN)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		slog.Info("Tip: Start Postgres with 'docker compose up -d postgres' or ensure Postgres is running")
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("Successfully connected to PostgreSQL database")

	// Initialize Redis client
	slog.Info("Connecting to Redis", "addr", cfg.RedisAddr)
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})
	defer redisClient.Close()

	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
		slog.Info("Tip: Start Redis with 'docker compose up -d redis' or ensure Redis is running")
		os.Exit(1)
	}
	slog.Info("Successfully connected to Redis")

	// Initialize snapshot writer
	snapshotWriter := snapshot.NewWriter(redisClient)

	// Build initial snapshot from all enabled rules
	slog.Info("Building initial snapshot from all enabled rules")
	if err := rebuildSnapshot(ctx, db, snapshotWriter); err != nil {
		slog.Error("Failed to build initial snapshot", "error", err)
		os.Exit(1)
	}

	// Initialize Kafka consumer
	slog.Info("Connecting to Kafka consumer", "topic", cfg.RuleChangedTopic)
	kafkaConsumer, err := consumer.NewConsumer(cfg.KafkaBrokers, cfg.RuleChangedTopic, cfg.ConsumerGroupID)
	if err != nil {
		slog.Error("Failed to create Kafka consumer", "error", err)
		slog.Info("Tip: Start Kafka with 'docker compose up -d kafka'")
		os.Exit(1)
	}
	defer kafkaConsumer.Close()
	slog.Info("Successfully connected to Kafka consumer")

	// Initialize processor
	proc := processor.NewProcessor(kafkaConsumer, db, snapshotWriter)

	// Main processing loop: consume rule.changed events and rebuild snapshot
	slog.Info("Starting rule.changed event processing loop")
	if err := proc.ProcessRuleChanges(ctx); err != nil {
		slog.Error("Rule change processing failed", "error", err)
		os.Exit(1)
	}

	slog.Info("Rule-updater service stopped")
}

// rebuildSnapshot queries all enabled rules from the database, builds a snapshot,
// and writes it to Redis with an incremented version.
func rebuildSnapshot(ctx context.Context, db *database.DB, writer *snapshot.Writer) error {
	// Query all enabled rules
	rules, err := db.GetAllEnabledRules(ctx)
	if err != nil {
		return err
	}

	slog.Info("Found enabled rules", "count", len(rules))

	// Build snapshot from rules
	snap := snapshot.BuildSnapshot(rules)

	// Write snapshot to Redis (this also increments the version)
	if err := writer.WriteSnapshot(ctx, snap); err != nil {
		return err
	}

	slog.Info("Snapshot rebuilt successfully",
		"rules_count", len(rules),
		"severity_dict_size", len(snap.SeverityDict),
		"source_dict_size", len(snap.SourceDict),
		"name_dict_size", len(snap.NameDict),
	)

	return nil
}

// getEnvOrDefault returns the environment variable value or a default value if not set.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// maskDSN masks sensitive information in the DSN for logging.
func maskDSN(dsn string) string {
	// Simple masking: replace password with ***
	// This is a basic implementation - in production, use a proper DSN parser
	if len(dsn) > 50 {
		return dsn[:20] + "***" + dsn[len(dsn)-20:]
	}
	return "***"
}
