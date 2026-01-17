.PHONY: check-migrations list-migrations migration-status help setup-infra verify-deps run-migrations create-topics generate-test-data run-all run-all-bg run-producer run-single-test stop-services stop-infra stop-all

help:
	@echo "Infrastructure Management:"
	@echo "  setup-infra       - Start all shared infrastructure (Postgres, Kafka, Redis, Zookeeper, MailHog)"
	@echo "  verify-deps       - Verify all dependencies are running and accessible"
	@echo "  create-topics     - Create all Kafka topics used by services"
	@echo "  run-migrations    - Run all migrations from all services (centralized)"
	@echo ""
	@echo "Service Management:"
	@echo "  run-all           - Run all services (in separate terminals or background)"
	@echo "  run-all-bg        - Run all services in background mode"
	@echo "  run-producer      - Run alert-producer in a separate terminal (on-demand)"
	@echo "  run-single-test   - Send a single test alert (LOW/test-source/test-name) via alert-producer"
	@echo "  stop-services     - Stop all application services"
	@echo "  stop-infra        - Stop all infrastructure (Postgres, Kafka, Redis, Zookeeper, MailHog)"
	@echo "  stop-all          - Stop both services and infrastructure"
	@echo ""
	@echo "Migration Management:"
	@echo "  check-migrations  - Validate migration consistency across services"
	@echo "  list-migrations   - List all migrations across services"
	@echo "  migration-status  - Show current database migration status"
	@echo ""
	@echo "Test Data:"
	@echo "  generate-test-data - Clean database and generate 100 clients with rules and endpoints"

# Infrastructure management
setup-infra:
	@./scripts/setup-infrastructure.sh

verify-deps:
	@./scripts/verify-dependencies.sh

run-migrations:
	@./scripts/run-migrations.sh

# Check migration consistency across all services
check-migrations:
	@echo "Checking migration consistency across services..."
	@echo ""
	@echo "=== Migration Files ==="
	@find . -path "*/services/*/migrations/*.sql" -type f | sort | while read file; do \
		service=$$(echo $$file | sed 's|^\./.*/services/\([^/]*\)/.*|\1|'); \
		version=$$(basename $$file | sed 's/^\([0-9]*\)_.*/\1/'); \
		direction=$$(basename $$file | sed 's/.*\.\(up\|down\)\.sql/\1/'); \
		printf "%-15s %-6s %-4s %s\n" "$$service" "$$version" "$$direction" "$$(basename $$file)"; \
	done
	@echo ""
	@echo "=== Version Conflicts ==="
	@versions=$$(find . -path "*/services/*/migrations/*.up.sql" | sed 's/.*\/\([0-9]*\)_.*/\1/' | sort -n); \
	prev=""; \
	for v in $$versions; do \
		if [ "$$v" = "$$prev" ]; then \
			echo "⚠️  CONFLICT: Version $$v is used by multiple services!"; \
		fi; \
		prev=$$v; \
	done
	@echo ""
	@echo "=== Missing Down Migrations ==="
	@find . -path "*/services/*/migrations/*.up.sql" | while read upfile; do \
		downfile=$$(echo $$upfile | sed 's/\.up\.sql$$/.down.sql/'); \
		if [ ! -f "$$downfile" ]; then \
			echo "⚠️  Missing down migration for: $$upfile"; \
		fi; \
	done
	@echo ""
	@echo "=== Summary ==="
	@total=$$(find . -path "*/services/*/migrations/*.up.sql" | wc -l | tr -d ' '); \
	highest=$$(find . -path "*/services/*/migrations/*.up.sql" | sed 's/.*\/\([0-9]*\)_.*/\1/' | sort -n | tail -1); \
	echo "Total migrations: $$total"; \
	echo "Highest version: $$highest"; \
	echo ""
	@echo "✅ Migration check complete"

# List all migrations in a readable format
list-migrations:
	@echo "All migrations across services:"
	@echo ""
	@find . -path "*/services/*/migrations/*.up.sql" -type f | sort | while read file; do \
		service=$$(echo $$file | sed 's|^\./.*/services/\([^/]*\)/.*|\1|'); \
		version=$$(basename $$file | sed 's/^\([0-9]*\)_.*/\1/'); \
		name=$$(basename $$file | sed 's/^[0-9]*_\(.*\)\.up\.sql/\1/'); \
		printf "%-6s %-20s %s\n" "$$version" "$$service" "$$name"; \
	done | column -t

# Show current database migration status
migration-status:
	@echo "Current database migration status:"
	@echo ""
	@if docker ps --format "{{.Names}}" | grep -q "alerting-platform-postgres"; then \
		POSTGRES_CONTAINER="alerting-platform-postgres"; \
		echo "Using Postgres container: $$POSTGRES_CONTAINER"; \
		docker exec $$POSTGRES_CONTAINER psql -U postgres -d alerting -c "SELECT version, dirty FROM schema_migrations;" 2>/dev/null || echo "Could not query migrations table (may not exist yet)"; \
	elif docker ps --format "{{.Names}}" | grep -q "postgres"; then \
		POSTGRES_CONTAINER=$$(docker ps --format "{{.Names}}" | grep postgres | head -1); \
		echo "Using Postgres container: $$POSTGRES_CONTAINER"; \
		docker exec $$POSTGRES_CONTAINER psql -U postgres -d alerting -c "SELECT version, dirty FROM schema_migrations;" 2>/dev/null || echo "Could not query migrations table (may not exist yet)"; \
	else \
		echo "⚠️  No Postgres container found. Start infrastructure with: make setup-infra"; \
	fi

# Create all Kafka topics
create-topics:
	@./scripts/create-kafka-topics.sh

# Generate test data (100 clients with rules and endpoints)
generate-test-data:
	@./scripts/test-data/generate-test-data.sh

# Run all services
run-all:
	@./scripts/run-all-services.sh

# Run all services in background
run-all-bg:
	@./scripts/run-all-services.sh --background

# Run alert-producer in a separate terminal (on-demand)
run-producer:
	@./scripts/run-single-service.sh alert-producer

# Send a single test alert (LOW/test-source/test-name)
run-single-test:
	@./scripts/run-single-test.sh

# Stop all application services
stop-services:
	@./scripts/stop-all-services.sh

# Stop infrastructure
stop-infra:
	@./scripts/stop-infrastructure.sh

# Stop everything (services + infrastructure)
stop-all:
	@./scripts/stop-all-services.sh
	@echo ""
	@./scripts/stop-infrastructure.sh
