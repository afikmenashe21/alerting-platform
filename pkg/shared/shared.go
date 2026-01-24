// Package shared provides common utility functions used across services.
package shared

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

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
