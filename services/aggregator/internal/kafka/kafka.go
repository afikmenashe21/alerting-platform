// Package kafka provides shared Kafka utilities for the aggregator service.
package kafka

import "strings"

// ParseBrokers parses a comma-separated broker string into a slice of broker addresses.
// Trims whitespace from each broker address.
func ParseBrokers(brokers string) []string {
	brokerList := strings.Split(brokers, ",")
	for i := range brokerList {
		brokerList[i] = strings.TrimSpace(brokerList[i])
	}
	return brokerList
}
