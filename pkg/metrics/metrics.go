// Package metrics provides a shared metrics collection and reporting system.
// Services write metrics to Redis for centralized access.
package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// MetricsKeyPrefix is the Redis key prefix for service metrics.
	MetricsKeyPrefix = "metrics:"
	// MetricsTTL is how long metrics stay in Redis if not refreshed.
	MetricsTTL = 2 * time.Minute
	// DefaultReportInterval is the default interval for writing metrics to Redis.
	DefaultReportInterval = 30 * time.Second
)

// ServiceMetrics holds metrics for a single service.
type ServiceMetrics struct {
	ServiceName string    `json:"service_name"`
	StartedAt   time.Time `json:"started_at"`
	LastUpdated time.Time `json:"last_updated"`
	Status      string    `json:"status"` // "healthy" or "unhealthy"

	// Counters (monotonically increasing since start)
	MessagesReceived  uint64 `json:"messages_received"`
	MessagesProcessed uint64 `json:"messages_processed"`
	MessagesPublished uint64 `json:"messages_published"`
	ProcessingErrors  uint64 `json:"processing_errors"`

	// Rates (per report interval)
	MessagesPerSecond float64 `json:"messages_per_second"`

	// Latencies (averages in nanoseconds)
	AvgProcessingLatencyNs float64 `json:"avg_processing_latency_ns"`

	// Service-specific counters (flexible map)
	CustomCounters map[string]uint64 `json:"custom_counters,omitempty"`
}

// Collector collects and reports metrics for a service.
type Collector struct {
	serviceName    string
	redis          *redis.Client
	startedAt      time.Time
	reportInterval time.Duration

	// Atomic counters
	messagesReceived  atomic.Uint64
	messagesProcessed atomic.Uint64
	messagesPublished atomic.Uint64
	processingErrors  atomic.Uint64

	// For rate calculation
	lastReportTime     time.Time
	lastProcessedCount uint64

	// Latency tracking
	totalLatencyNs atomic.Uint64
	latencyCount   atomic.Uint64

	// Custom counters
	customMu       sync.RWMutex
	customCounters map[string]*atomic.Uint64

	// Stop channel
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewCollector creates a new metrics collector for a service.
func NewCollector(serviceName string, redisClient *redis.Client) *Collector {
	return &Collector{
		serviceName:    serviceName,
		redis:          redisClient,
		startedAt:      time.Now().UTC(),
		reportInterval: DefaultReportInterval,
		lastReportTime: time.Now().UTC(),
		customCounters: make(map[string]*atomic.Uint64),
		stopCh:         make(chan struct{}),
	}
}

// SetReportInterval sets the interval for writing metrics to Redis.
func (c *Collector) SetReportInterval(interval time.Duration) {
	c.reportInterval = interval
}

// Start begins the periodic metrics reporting to Redis.
func (c *Collector) Start(ctx context.Context) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		ticker := time.NewTicker(c.reportInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				c.writeMetrics(context.Background()) // Final write
				return
			case <-c.stopCh:
				c.writeMetrics(context.Background()) // Final write
				return
			case <-ticker.C:
				c.writeMetrics(ctx)
			}
		}
	}()
}

// Stop stops the metrics reporting.
func (c *Collector) Stop() {
	close(c.stopCh)
	c.wg.Wait()
}

// RecordReceived increments the messages received counter.
func (c *Collector) RecordReceived() {
	c.messagesReceived.Add(1)
}

// RecordProcessed increments the messages processed counter with latency.
func (c *Collector) RecordProcessed(latency time.Duration) {
	c.messagesProcessed.Add(1)
	c.totalLatencyNs.Add(uint64(latency.Nanoseconds()))
	c.latencyCount.Add(1)
}

// RecordPublished increments the messages published counter.
func (c *Collector) RecordPublished() {
	c.messagesPublished.Add(1)
}

// RecordError increments the processing errors counter.
func (c *Collector) RecordError() {
	c.processingErrors.Add(1)
}

// IncrementCustom increments a custom counter by name.
func (c *Collector) IncrementCustom(name string) {
	c.customMu.RLock()
	counter, exists := c.customCounters[name]
	c.customMu.RUnlock()

	if !exists {
		c.customMu.Lock()
		// Double-check after acquiring write lock
		if counter, exists = c.customCounters[name]; !exists {
			counter = &atomic.Uint64{}
			c.customCounters[name] = counter
		}
		c.customMu.Unlock()
	}
	counter.Add(1)
}

// AddCustom adds a value to a custom counter.
func (c *Collector) AddCustom(name string, value uint64) {
	c.customMu.RLock()
	counter, exists := c.customCounters[name]
	c.customMu.RUnlock()

	if !exists {
		c.customMu.Lock()
		if counter, exists = c.customCounters[name]; !exists {
			counter = &atomic.Uint64{}
			c.customCounters[name] = counter
		}
		c.customMu.Unlock()
	}
	counter.Add(value)
}

// GetSnapshot returns current metrics without writing to Redis.
func (c *Collector) GetSnapshot() *ServiceMetrics {
	now := time.Now().UTC()
	processed := c.messagesProcessed.Load()

	// Calculate rate
	elapsed := now.Sub(c.lastReportTime).Seconds()
	var rate float64
	if elapsed > 0 {
		rate = float64(processed-c.lastProcessedCount) / elapsed
	}

	// Calculate average latency in nanoseconds
	var avgLatencyNs float64
	latencyCount := c.latencyCount.Load()
	if latencyCount > 0 {
		avgLatencyNs = float64(c.totalLatencyNs.Load()) / float64(latencyCount)
	}

	// Build custom counters map
	c.customMu.RLock()
	customCounters := make(map[string]uint64, len(c.customCounters))
	for name, counter := range c.customCounters {
		customCounters[name] = counter.Load()
	}
	c.customMu.RUnlock()

	return &ServiceMetrics{
		ServiceName:            c.serviceName,
		StartedAt:              c.startedAt,
		LastUpdated:            now,
		Status:                 "healthy",
		MessagesReceived:       c.messagesReceived.Load(),
		MessagesProcessed:      processed,
		MessagesPublished:      c.messagesPublished.Load(),
		ProcessingErrors:       c.processingErrors.Load(),
		MessagesPerSecond:      rate,
		AvgProcessingLatencyNs: avgLatencyNs,
		CustomCounters:         customCounters,
	}
}

// writeMetrics writes current metrics to Redis.
func (c *Collector) writeMetrics(ctx context.Context) {
	if c.redis == nil {
		return
	}

	metrics := c.GetSnapshot()

	// Update rate calculation state
	c.lastReportTime = metrics.LastUpdated
	c.lastProcessedCount = metrics.MessagesProcessed

	// Note: We do NOT reset latency counters - we want all-time average latency
	// This ensures latency is visible even after burst processing completes

	data, err := json.Marshal(metrics)
	if err != nil {
		slog.Error("Failed to marshal metrics", "service", c.serviceName, "error", err)
		return
	}

	key := MetricsKeyPrefix + c.serviceName
	if err := c.redis.Set(ctx, key, data, MetricsTTL).Err(); err != nil {
		slog.Error("Failed to write metrics to Redis", "service", c.serviceName, "error", err)
		return
	}

	slog.Debug("Metrics written to Redis", "service", c.serviceName, "key", key)
}

// Reader reads service metrics from Redis.
type Reader struct {
	redis *redis.Client
}

// NewReader creates a new metrics reader.
func NewReader(redisClient *redis.Client) *Reader {
	return &Reader{redis: redisClient}
}

// GetServiceMetrics retrieves metrics for a specific service.
func (r *Reader) GetServiceMetrics(ctx context.Context, serviceName string) (*ServiceMetrics, error) {
	key := MetricsKeyPrefix + serviceName
	data, err := r.redis.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("no metrics found for service: %s", serviceName)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read metrics: %w", err)
	}

	var metrics ServiceMetrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metrics: %w", err)
	}

	// Check if metrics are stale (older than TTL)
	if time.Since(metrics.LastUpdated) > MetricsTTL {
		metrics.Status = "unhealthy"
	}

	return &metrics, nil
}

// GetAllServiceMetrics retrieves metrics for all services.
func (r *Reader) GetAllServiceMetrics(ctx context.Context) (map[string]*ServiceMetrics, error) {
	pattern := MetricsKeyPrefix + "*"
	keys, err := r.redis.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list metrics keys: %w", err)
	}

	result := make(map[string]*ServiceMetrics)
	for _, key := range keys {
		serviceName := key[len(MetricsKeyPrefix):]
		metrics, err := r.GetServiceMetrics(ctx, serviceName)
		if err != nil {
			slog.Warn("Failed to read metrics for service", "service", serviceName, "error", err)
			continue
		}
		result[serviceName] = metrics
	}

	return result, nil
}

// ServiceNames is the list of known services for the UI.
var ServiceNames = []string{
	"alert-producer",
	"evaluator",
	"aggregator",
	"sender",
	"rule-service",
	"rule-updater",
}

// --- Centralized Helper Functions ---
// These helpers reduce code duplication across services.

// GetEnvOrDefault returns the environment variable value or a default if not set.
func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// MaskDSN masks sensitive information in a DSN for logging.
func MaskDSN(dsn string) string {
	if len(dsn) > 50 {
		return dsn[:20] + "***" + dsn[len(dsn)-20:]
	}
	return "***"
}

// ConnectRedis creates and validates a Redis connection.
// Returns the client and nil on success, or nil and an error on failure.
func ConnectRedis(ctx context.Context, addr string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to connect to Redis at %s: %w", addr, err)
	}

	return client, nil
}
