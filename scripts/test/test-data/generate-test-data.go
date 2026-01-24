package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

const (
	defaultDSN = "postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable"

	// Target: 90% of evaluator memory capacity (~500k rules max → 450k target)
	numClients      = 1500
	rulesPerClient  = 300 // 94% of 320 max per client (4 severities × 8 sources × 10 names)
	endpointsPerRule = 2

	// Batch insert size for performance on remote RDS
	batchSize = 1000
)

var (
	severities    = []string{"LOW", "MEDIUM", "HIGH", "CRITICAL"}
	sources       = []string{"api", "db", "cache", "monitor", "queue", "worker", "frontend", "backend"}
	names         = []string{"timeout", "error", "crash", "slow", "memory", "cpu", "disk", "network", "auth", "validation"}
	endpointTypes = []string{"email", "webhook", "slack"}
)

// ruleCombination represents a unique (severity, source, name) tuple
type ruleCombination struct {
	severity string
	source   string
	name     string
}

func main() {
	dsn := defaultDSN
	if envDSN := os.Getenv("POSTGRES_DSN"); envDSN != "" {
		dsn = envDSN
	}
	if len(os.Args) > 1 {
		dsn = os.Args[1]
	}

	log.Printf("Connecting to database...")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Increase connection pool for batch operations
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Pre-generate all 320 possible rule combinations
	allCombinations := generateAllCombinations()
	log.Printf("Possible rule combinations per client: %d", len(allCombinations))
	log.Printf("Using %d combinations per client (%.0f%% fill)", rulesPerClient, float64(rulesPerClient)/float64(len(allCombinations))*100)

	log.Printf("")
	log.Printf("=== Target Data ===")
	log.Printf("Clients:   %d", numClients)
	log.Printf("Rules:     %d (%d clients × %d rules)", numClients*rulesPerClient, numClients, rulesPerClient)
	log.Printf("Endpoints: %d (%d rules × %d endpoints)", numClients*rulesPerClient*endpointsPerRule, numClients*rulesPerClient, endpointsPerRule)
	log.Printf("")

	start := time.Now()

	log.Printf("Cleaning database...")
	if err := cleanDatabase(ctx, db); err != nil {
		log.Fatalf("Failed to clean database: %v", err)
	}

	log.Printf("Creating %d clients...", numClients)
	if err := createClients(ctx, db); err != nil {
		log.Fatalf("Failed to create clients: %v", err)
	}

	log.Printf("Creating %d rules (batch size: %d)...", numClients*rulesPerClient, batchSize)
	ruleIDs, err := createRules(ctx, db, allCombinations)
	if err != nil {
		log.Fatalf("Failed to create rules: %v", err)
	}

	log.Printf("Creating %d endpoints (batch size: %d)...", len(ruleIDs)*endpointsPerRule, batchSize)
	endpointsCreated, err := createEndpoints(ctx, db, ruleIDs)
	if err != nil {
		log.Fatalf("Failed to create endpoints: %v", err)
	}

	elapsed := time.Since(start)
	log.Printf("")
	log.Printf("=== Generation Complete ===")
	log.Printf("Clients:   %d", numClients)
	log.Printf("Rules:     %d", len(ruleIDs))
	log.Printf("Endpoints: %d", endpointsCreated)
	log.Printf("Duration:  %s", elapsed.Round(time.Second))
	log.Printf("")
	log.Printf("Evaluator memory estimate: ~%.0f MB (of 100 MB budget)", float64(len(ruleIDs))*200/1024/1024)
	log.Printf("Redis snapshot estimate:   ~%.0f MB (of 460 MB budget)", float64(len(ruleIDs))*150/1024/1024)
}

func generateAllCombinations() []ruleCombination {
	combinations := make([]ruleCombination, 0, len(severities)*len(sources)*len(names))
	for _, sev := range severities {
		for _, src := range sources {
			for _, name := range names {
				combinations = append(combinations, ruleCombination{sev, src, name})
			}
		}
	}
	return combinations
}

func cleanDatabase(ctx context.Context, db *sql.DB) error {
	queries := []string{
		"DELETE FROM endpoints",
		"DELETE FROM rules",
		"DELETE FROM notifications",
		"DELETE FROM clients",
	}
	for _, query := range queries {
		if _, err := db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to execute %s: %w", query, err)
		}
	}
	return nil
}

func createClients(ctx context.Context, db *sql.DB) error {
	for batchStart := 1; batchStart <= numClients; batchStart += batchSize {
		batchEnd := batchStart + batchSize - 1
		if batchEnd > numClients {
			batchEnd = numClients
		}

		values := make([]string, 0, batchEnd-batchStart+1)
		args := make([]interface{}, 0, (batchEnd-batchStart+1)*2)
		argIdx := 1

		for i := batchStart; i <= batchEnd; i++ {
			clientID := fmt.Sprintf("client-%05d", i)
			clientName := fmt.Sprintf("Client %d", i)
			values = append(values, fmt.Sprintf("($%d, $%d, NOW(), NOW())", argIdx, argIdx+1))
			args = append(args, clientID, clientName)
			argIdx += 2
		}

		query := fmt.Sprintf(
			"INSERT INTO clients (client_id, name, created_at, updated_at) VALUES %s ON CONFLICT (client_id) DO NOTHING",
			strings.Join(values, ", "),
		)
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("batch insert clients %d-%d: %w", batchStart, batchEnd, err)
		}

		if batchEnd%5000 == 0 || batchEnd == numClients {
			log.Printf("  Clients: %d/%d", batchEnd, numClients)
		}
	}
	return nil
}

func createRules(ctx context.Context, db *sql.DB, combinations []ruleCombination) ([]string, error) {
	allRuleIDs := make([]string, 0, numClients*rulesPerClient)
	totalCreated := 0

	for clientIdx := 1; clientIdx <= numClients; clientIdx++ {
		clientID := fmt.Sprintf("client-%05d", clientIdx)

		// Pick the first rulesPerClient combinations for this client (deterministic, no conflicts)
		clientCombinations := combinations[:rulesPerClient]

		// Batch insert rules for this client
		for batchStart := 0; batchStart < len(clientCombinations); batchStart += batchSize {
			batchEnd := batchStart + batchSize
			if batchEnd > len(clientCombinations) {
				batchEnd = len(clientCombinations)
			}

			batch := clientCombinations[batchStart:batchEnd]
			values := make([]string, 0, len(batch))
			args := make([]interface{}, 0, len(batch)*4)
			argIdx := 1

			for _, combo := range batch {
				values = append(values, fmt.Sprintf("($%d, $%d, $%d, $%d, TRUE, 1, NOW(), NOW())", argIdx, argIdx+1, argIdx+2, argIdx+3))
				args = append(args, clientID, combo.severity, combo.source, combo.name)
				argIdx += 4
			}

			query := fmt.Sprintf(
				"INSERT INTO rules (client_id, severity, source, name, enabled, version, created_at, updated_at) VALUES %s ON CONFLICT (client_id, severity, source, name) DO NOTHING RETURNING rule_id",
				strings.Join(values, ", "),
			)

			rows, err := db.QueryContext(ctx, query, args...)
			if err != nil {
				return nil, fmt.Errorf("batch insert rules for %s: %w", clientID, err)
			}

			for rows.Next() {
				var ruleID string
				if err := rows.Scan(&ruleID); err != nil {
					rows.Close()
					return nil, fmt.Errorf("scan rule_id: %w", err)
				}
				allRuleIDs = append(allRuleIDs, ruleID)
				totalCreated++
			}
			rows.Close()
		}

		if clientIdx%100 == 0 || clientIdx == numClients {
			log.Printf("  Rules: %d/%d (client %d/%d)", totalCreated, numClients*rulesPerClient, clientIdx, numClients)
		}
	}

	return allRuleIDs, nil
}

func createEndpoints(ctx context.Context, db *sql.DB, ruleIDs []string) (int, error) {
	totalCreated := 0

	for batchStart := 0; batchStart < len(ruleIDs); batchStart += batchSize / endpointsPerRule {
		batchEnd := batchStart + batchSize/endpointsPerRule
		if batchEnd > len(ruleIDs) {
			batchEnd = len(ruleIDs)
		}

		values := make([]string, 0, (batchEnd-batchStart)*endpointsPerRule)
		args := make([]interface{}, 0, (batchEnd-batchStart)*endpointsPerRule*3)
		argIdx := 1

		for i := batchStart; i < batchEnd; i++ {
			ruleID := ruleIDs[i]

			// Create 2 endpoints per rule: email + webhook
			emailValue := fmt.Sprintf("alert-%d@example.com", i+1)
			values = append(values, fmt.Sprintf("($%d, 'email', $%d, TRUE, NOW(), NOW())", argIdx, argIdx+1))
			args = append(args, ruleID, emailValue)
			argIdx += 2

			webhookValue := fmt.Sprintf("https://webhook.example.com/rule/%s", ruleID[:8])
			values = append(values, fmt.Sprintf("($%d, 'webhook', $%d, TRUE, NOW(), NOW())", argIdx, argIdx+1))
			args = append(args, ruleID, webhookValue)
			argIdx += 2
		}

		query := fmt.Sprintf(
			"INSERT INTO endpoints (rule_id, type, value, enabled, created_at, updated_at) VALUES %s ON CONFLICT (rule_id, type, value) DO NOTHING",
			strings.Join(values, ", "),
		)
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			return 0, fmt.Errorf("batch insert endpoints at offset %d: %w", batchStart, err)
		}

		totalCreated += (batchEnd - batchStart) * endpointsPerRule

		if totalCreated%(batchSize*10) == 0 || batchEnd == len(ruleIDs) {
			log.Printf("  Endpoints: %d/%d", totalCreated, len(ruleIDs)*endpointsPerRule)
		}
	}

	return totalCreated, nil
}
