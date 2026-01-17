package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"evaluator/internal/config"
	"evaluator/internal/consumer"
	"evaluator/internal/indexes"
	"evaluator/internal/matcher"
	"evaluator/internal/processor"
	"evaluator/internal/producer"
	"evaluator/internal/reloader"
	"evaluator/internal/ruleconsumer"
	"evaluator/internal/snapshot"

	"github.com/redis/go-redis/v9"
)

func main() {
	// Parse command-line flags
	cfg := &config.Config{}
	flag.StringVar(&cfg.KafkaBrokers, "kafka-brokers", "localhost:9092", "Kafka broker addresses (comma-separated)")
	flag.StringVar(&cfg.AlertsNewTopic, "alerts-new-topic", "alerts.new", "Kafka topic for incoming alerts")
	flag.StringVar(&cfg.AlertsMatchedTopic, "alerts-matched-topic", "alerts.matched", "Kafka topic for matched alerts")
	flag.StringVar(&cfg.RuleChangedTopic, "rule-changed-topic", "rule.changed", "Kafka topic for rule change events")
	flag.StringVar(&cfg.ConsumerGroupID, "consumer-group-id", "evaluator-group", "Kafka consumer group ID for alerts.new")
	flag.StringVar(&cfg.RuleChangedGroupID, "rule-changed-group-id", "evaluator-rule-changed-group", "Kafka consumer group ID for rule.changed")
	flag.StringVar(&cfg.RedisAddr, "redis-addr", "localhost:6379", "Redis server address")
	flag.DurationVar(&cfg.VersionPollInterval, "version-poll-interval", 5*time.Second, "Interval for polling Redis version")
	flag.Parse()

	// Set up structured logging
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	slog.Info("Starting evaluator service",
		"kafka_brokers", cfg.KafkaBrokers,
		"alerts_new_topic", cfg.AlertsNewTopic,
		"alerts_matched_topic", cfg.AlertsMatchedTopic,
		"rule_changed_topic", cfg.RuleChangedTopic,
		"consumer_group_id", cfg.ConsumerGroupID,
		"rule_changed_group_id", cfg.RuleChangedGroupID,
		"redis_addr", cfg.RedisAddr,
		"version_poll_interval", cfg.VersionPollInterval,
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

	// Initialize snapshot loader
	loader := snapshot.NewLoader(redisClient)

	// Load initial snapshot
	slog.Info("Loading initial rule snapshot from Redis")
	snap, err := loader.LoadSnapshot(ctx)
	if err != nil {
		slog.Error("Failed to load initial snapshot", "error", err)
		slog.Info("Tip: Ensure rule-updater has created the snapshot in Redis")
		os.Exit(1)
	}

	// Build initial indexes
	initialIndexes := indexes.NewIndexes(snap)
	ruleMatcher := matcher.NewMatcher(initialIndexes)
	slog.Info("Initial indexes built",
		"rules_count", initialIndexes.RuleCount(),
	)

	// Start version reloader (polls Redis for version changes)
	reload := reloader.NewReloader(loader, ruleMatcher, cfg.VersionPollInterval)
	if err := reload.Start(ctx); err != nil {
		slog.Error("Failed to start version reloader", "error", err)
		os.Exit(1)
	}

	// Initialize rule.changed consumer (for immediate rule updates)
	slog.Info("Connecting to rule.changed consumer", "topic", cfg.RuleChangedTopic)
	ruleChangedConsumer, err := ruleconsumer.NewConsumer(cfg.KafkaBrokers, cfg.RuleChangedTopic, cfg.RuleChangedGroupID)
	if err != nil {
		slog.Error("Failed to create rule.changed consumer", "error", err)
		slog.Info("Tip: Start Kafka with 'docker compose up -d kafka'")
		os.Exit(1)
	}
	defer ruleChangedConsumer.Close()
	slog.Info("Successfully connected to rule.changed consumer")

	// Initialize rule change handler
	ruleHandler := processor.NewRuleHandler(ruleChangedConsumer, reload)
	go ruleHandler.HandleRuleChanged(ctx)

	// Initialize Kafka consumer
	slog.Info("Connecting to Kafka consumer", "topic", cfg.AlertsNewTopic)
	kafkaConsumer, err := consumer.NewConsumer(cfg.KafkaBrokers, cfg.AlertsNewTopic, cfg.ConsumerGroupID)
	if err != nil {
		slog.Error("Failed to create Kafka consumer", "error", err)
		slog.Info("Tip: Start Kafka with 'docker compose up -d kafka'")
		os.Exit(1)
	}
	defer kafkaConsumer.Close()
	slog.Info("Successfully connected to Kafka consumer")

	// Initialize Kafka producer
	slog.Info("Connecting to Kafka producer", "topic", cfg.AlertsMatchedTopic)
	kafkaProducer, err := producer.NewProducer(cfg.KafkaBrokers, cfg.AlertsMatchedTopic)
	if err != nil {
		slog.Error("Failed to create Kafka producer", "error", err)
		os.Exit(1)
	}
	defer kafkaProducer.Close()
	slog.Info("Successfully connected to Kafka producer")

	// Initialize processor
	proc := processor.NewProcessor(kafkaConsumer, kafkaProducer, ruleMatcher)

	// Main processing loop
	slog.Info("Starting alert evaluation loop")
	if err := proc.ProcessAlerts(ctx); err != nil {
		slog.Error("Alert processing failed", "error", err)
		os.Exit(1)
	}

	slog.Info("Evaluator service stopped")
}

