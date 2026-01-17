package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	_ "github.com/lib/pq"
)

const (
	defaultDSN = "postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable"
)

var (
	severities    = []string{"LOW", "MEDIUM", "HIGH", "CRITICAL"}
	sources       = []string{"api", "db", "cache", "monitor", "queue", "worker", "frontend", "backend"}
	names         = []string{"timeout", "error", "crash", "slow", "memory", "cpu", "disk", "network", "auth", "validation"}
	endpointTypes = []string{"email", "webhook", "slack"}
)

func main() {
	dsn := defaultDSN
	if len(os.Args) > 1 {
		dsn = os.Args[1]
	}

	log.Printf("Connecting to database...")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Printf("Cleaning database...")
	if err := cleanDatabase(ctx, db); err != nil {
		log.Fatalf("Failed to clean database: %v", err)
	}

	log.Printf("Generating 100 clients with rules and endpoints...")
	rand.Seed(time.Now().UnixNano())

	clientsCreated := 0
	rulesCreated := 0
	endpointsCreated := 0

	for i := 1; i <= 100; i++ {
		clientID := fmt.Sprintf("client-%03d", i)
		clientName := fmt.Sprintf("Client %d", i)

		// Create client
		if err := createClient(ctx, db, clientID, clientName); err != nil {
			log.Printf("Warning: Failed to create client %s: %v", clientID, err)
			continue
		}
		clientsCreated++

		// Generate 1-5 rules per client (random distribution)
		numRules := rand.Intn(5) + 1
		for j := 0; j < numRules; j++ {
			severity := severities[rand.Intn(len(severities))]
			source := sources[rand.Intn(len(sources))]
			name := names[rand.Intn(len(names))]

			ruleID, err := createRule(ctx, db, clientID, severity, source, name)
			if err != nil {
				log.Printf("Warning: Failed to create rule for client %s: %v", clientID, err)
				continue
			}
			rulesCreated++

			// Generate 1-3 endpoints per rule (random distribution)
			numEndpoints := rand.Intn(3) + 1
			endpointTypesUsed := make(map[string]bool)

			for k := 0; k < numEndpoints; k++ {
				// Ensure we don't create duplicate endpoint types for the same rule
				endpointType := endpointTypes[rand.Intn(len(endpointTypes))]
				maxAttempts := 10
				for endpointTypesUsed[endpointType] && maxAttempts > 0 {
					endpointType = endpointTypes[rand.Intn(len(endpointTypes))]
					maxAttempts--
				}
				endpointTypesUsed[endpointType] = true

				var value string
				switch endpointType {
				case "email":
					value = fmt.Sprintf("alert-%03d-%d@example.com", i, k+1)
				case "webhook":
					value = fmt.Sprintf("https://webhook.example.com/client-%03d/rule-%s", i, ruleID[:8])
				case "slack":
					value = fmt.Sprintf("#alerts-client-%03d", i)
				}

				if err := createEndpoint(ctx, db, ruleID, endpointType, value); err != nil {
					log.Printf("Warning: Failed to create endpoint for rule %s: %v", ruleID, err)
					continue
				}
				endpointsCreated++
			}
		}

		if i%10 == 0 {
			log.Printf("Progress: %d clients, %d rules, %d endpoints created...", clientsCreated, rulesCreated, endpointsCreated)
		}
	}

	log.Printf("\n=== Generation Complete ===")
	log.Printf("Clients created: %d", clientsCreated)
	log.Printf("Rules created: %d", rulesCreated)
	log.Printf("Endpoints created: %d", endpointsCreated)
	log.Printf("Average rules per client: %.2f", float64(rulesCreated)/float64(clientsCreated))
	log.Printf("Average endpoints per rule: %.2f", float64(endpointsCreated)/float64(rulesCreated))
}

func cleanDatabase(ctx context.Context, db *sql.DB) error {
	// Delete in order: endpoints -> rules -> clients -> notifications
	// (respecting foreign key constraints)

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

func createClient(ctx context.Context, db *sql.DB, clientID, name string) error {
	query := `
		INSERT INTO clients (client_id, name, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (client_id) DO NOTHING
	`
	_, err := db.ExecContext(ctx, query, clientID, name)
	return err
}

func createRule(ctx context.Context, db *sql.DB, clientID, severity, source, name string) (string, error) {
	query := `
		INSERT INTO rules (client_id, severity, source, name, enabled, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, TRUE, 1, NOW(), NOW())
		ON CONFLICT (client_id, severity, source, name) DO NOTHING
		RETURNING rule_id
	`
	var ruleID string
	err := db.QueryRowContext(ctx, query, clientID, severity, source, name).Scan(&ruleID)
	if err == sql.ErrNoRows {
		// Rule already exists, fetch it
		query = `
			SELECT rule_id FROM rules
			WHERE client_id = $1 AND severity = $2 AND source = $3 AND name = $4
		`
		err = db.QueryRowContext(ctx, query, clientID, severity, source, name).Scan(&ruleID)
	}
	return ruleID, err
}

func createEndpoint(ctx context.Context, db *sql.DB, ruleID, endpointType, value string) error {
	query := `
		INSERT INTO endpoints (rule_id, type, value, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, TRUE, NOW(), NOW())
		ON CONFLICT (rule_id, type, value) DO NOTHING
	`
	_, err := db.ExecContext(ctx, query, ruleID, endpointType, value)
	return err
}
