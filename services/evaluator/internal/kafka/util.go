// Package kafka provides shared Kafka utilities for the evaluator service.
package kafka

import (
	"strings"
)

// ParseBrokers parses a comma-separated broker list and trims whitespace.
// Returns a slice of broker addresses.
func ParseBrokers(brokers string) []string {
	if brokers == "" {
		return nil
	}
	brokerList := strings.Split(brokers, ",")
	for i := range brokerList {
		brokerList[i] = strings.TrimSpace(brokerList[i])
	}
	return brokerList
}
