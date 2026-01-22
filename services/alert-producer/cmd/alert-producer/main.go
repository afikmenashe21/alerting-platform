// Package main provides the CLI entry point for the alert-producer service.
// It handles command-line flag parsing, service initialization, and delegates
// processing to the processor module.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"alert-producer/internal/config"
	"alert-producer/internal/generator"
	"alert-producer/internal/processor"
	"alert-producer/internal/producer"
)

func main() {
	// Initialize structured logger with JSON output
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg := config.Config{}
	var mockMode bool
	var testMode bool
	var singleTestMode bool
	flag.StringVar(&cfg.KafkaBrokers, "kafka-brokers", getEnvOrDefault("KAFKA_BROKERS", "localhost:9092"), "Kafka broker addresses (comma-separated)")
	flag.StringVar(&cfg.Topic, "topic", getEnvOrDefault("ALERTS_NEW_TOPIC", "alerts.new"), "Kafka topic name")
	flag.Float64Var(&cfg.RPS, "rps", 10.0, "Alerts per second")
	flag.DurationVar(&cfg.Duration, "duration", 60*time.Second, "Duration to run (e.g., 60s, 5m)")
	flag.IntVar(&cfg.BurstSize, "burst", 0, "Burst mode: send N alerts immediately, then stop (0 = continuous)")
	flag.Int64Var(&cfg.Seed, "seed", 0, "Random seed for deterministic generation (0 = random)")
	flag.StringVar(&cfg.SeverityDist, "severity-dist", "HIGH:30,MEDIUM:30,LOW:25,CRITICAL:15", "Severity distribution (format: SEVERITY:percent,...)")
	flag.StringVar(&cfg.SourceDist, "source-dist", "api:25,db:20,cache:15,monitor:15,queue:10,worker:5,frontend:5,backend:5", "Source distribution (format: source:percent,...)")
	flag.StringVar(&cfg.NameDist, "name-dist", "timeout:15,error:15,crash:10,slow:10,memory:10,cpu:10,disk:10,network:10,auth:5,validation:5", "Name distribution (format: name:percent,...)")
	flag.BoolVar(&mockMode, "mock", false, "Use mock producer (no Kafka required, logs alerts instead)")
	flag.BoolVar(&testMode, "test", false, "Test mode: generate test alert (LOW/test-source/test-name) matching afik-test rule")
	flag.BoolVar(&singleTestMode, "single-test", false, "Single test mode: send only one test alert (LOW/test-source/test-name) and exit")
	flag.Parse()

	slog.Info("Starting alert-producer",
		"kafka_brokers", cfg.KafkaBrokers,
		"topic", cfg.Topic,
		"rps", cfg.RPS,
		"duration", cfg.Duration,
		"burst_size", cfg.BurstSize,
		"seed", cfg.Seed,
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

	// Initialize producer (Kafka or Mock)
	var alertPublisher producer.AlertPublisher
	if mockMode {
		// Use mock producer (no Kafka required)
		slog.Info("Using mock mode - alerts will be logged but not sent to Kafka")
		alertPublisher = producer.NewMock(cfg.Topic)
		defer alertPublisher.Close()
	} else {
		// Use real Kafka producer
		slog.Info("Connecting to Kafka", "brokers", cfg.KafkaBrokers, "topic", cfg.Topic)
		kafkaProd, err := producer.New(cfg.KafkaBrokers, cfg.Topic)
		if err != nil {
			slog.Error("Failed to create Kafka producer", "error", err)
			slog.Info("Tip: Start Kafka with 'docker compose up -d' or use --mock flag to test without Kafka")
			os.Exit(1)
		}
		alertPublisher = kafkaProd
		defer alertPublisher.Close()
		slog.Info("Successfully connected to Kafka")
	}

	// Initialize alert generator
	gen := generator.New(cfg)
	slog.Info("Alert generator initialized",
		"severity_dist", cfg.SeverityDist,
		"source_dist", cfg.SourceDist,
		"name_dist", cfg.NameDist,
	)

	// Initialize processor
	proc := processor.NewProcessor(gen, alertPublisher, &cfg)

	// Handle single test mode - send only one test alert and exit
	if singleTestMode {
		slog.Info("Running in single test mode - sending one test alert (LOW/test-source/test-name)")
		testAlert := generator.GenerateTestAlert()
		if err := alertPublisher.Publish(ctx, testAlert); err != nil {
			slog.Error("Failed to publish test alert",
				"alert_id", testAlert.AlertID,
				"severity", testAlert.Severity,
				"source", testAlert.Source,
				"name", testAlert.Name,
				"error", err,
			)
			os.Exit(1)
		}
		alertJSON, _ := json.Marshal(testAlert)
		slog.Info("Successfully published single test alert",
			"alert_id", testAlert.AlertID,
			"severity", testAlert.Severity,
			"source", testAlert.Source,
			"name", testAlert.Name,
			"event_ts", testAlert.EventTS,
			"alert_json", string(alertJSON),
		)
		slog.Info("Single test mode completed successfully")
		return
	}

	// Handle test mode - generate varied alerts with one test alert included
	if testMode {
		slog.Info("Running in test mode - generating varied alerts with one test alert (LOW/test-source/test-name) included")
		if err := proc.ProcessTest(ctx, cfg.RPS, cfg.Duration, cfg.BurstSize); err != nil {
			slog.Error("Test mode failed", "error", err)
			os.Exit(1)
		}
		slog.Info("Test mode completed successfully")
		return
	}

	// Run normal processing mode
	if err := proc.Process(ctx); err != nil {
		slog.Error("Processing failed", "error", err)
		os.Exit(1)
	}

	slog.Info("Alert producer completed successfully")
}

// getEnvOrDefault returns the environment variable value or a default value if not set.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

