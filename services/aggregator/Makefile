.PHONY: build run test clean help deps migrate-up migrate-down migrate-create run-all

BINARY_NAME=aggregator
CMD_PATH=./cmd/aggregator

help:
	@echo "Available targets:"
	@echo "  run-all       - Verify dependencies, setup, and run aggregator (recommended)"
	@echo "  build         - Build the binary"
	@echo "  run           - Run the service"
	@echo "  test          - Run tests"
	@echo "  clean         - Remove build artifacts"
	@echo "  deps          - Download dependencies"
	@echo ""
	@echo "Database targets:"
	@echo "  db-query      - Show recent notifications (last 20)"
	@echo "  db-count      - Show notification counts by status"
	@echo "  db-all        - Show all notifications"
	@echo "  db-received   - Show notifications with status RECEIVED"
	@echo "  db-psql       - Open PostgreSQL interactive shell"
	@echo ""
	@echo "Migration targets:"
	@echo "  migrate-up    - Run database migrations (up)"
	@echo "  migrate-down  - Run database migrations (down)"
	@echo "  migrate-create - Create a new migration (usage: make migrate-create NAME=migration_name)"
	@echo "  check-migrations - Check migration consistency (from root: make check-migrations)"

build:
	@echo "Building $(BINARY_NAME)..."
	go build -o bin/$(BINARY_NAME) $(CMD_PATH)

run: build
	@echo "Running $(BINARY_NAME)..."
	./bin/$(BINARY_NAME) $(ARGS)

test:
	@echo "Running tests..."
	go test -v ./...

clean:
	@echo "Cleaning..."
	rm -rf bin/

deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Database migration targets
migrate-up:
	@echo "Running database migrations (up)..."
	@MIGRATE=$$(command -v migrate 2>/dev/null || [ -f ~/go/bin/migrate ] && echo ~/go/bin/migrate || [ -n "$$GOPATH" ] && [ -f $$GOPATH/bin/migrate ] && echo $$GOPATH/bin/migrate || echo ""); \
	if [ -z "$$MIGRATE" ]; then \
		echo "Error: migrate tool not found."; \
		echo "Install with: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"; \
		echo "If installed, ensure ~/go/bin is in your PATH or set GOPATH/bin in PATH"; \
		exit 1; \
	fi; \
	POSTGRES_DSN="$${POSTGRES_DSN:-postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable}"; \
	POSTGRES_CONTAINER=$$(docker ps --filter "name=postgres" --format "{{.Names}}" | head -n1 || echo ""); \
	if [ -n "$$POSTGRES_CONTAINER" ]; then \
		CURRENT_VERSION=$$(docker exec $$POSTGRES_CONTAINER psql -U postgres -d alerting -tAc "SELECT version FROM schema_migrations;" 2>/dev/null || echo "0"); \
		if [ -z "$$CURRENT_VERSION" ]; then CURRENT_VERSION="0"; fi; \
		if [ "$$CURRENT_VERSION" -ge 6 ]; then \
			echo "Database is already at version $$CURRENT_VERSION (aggregator migrations applied)"; \
		else \
			echo "Current version: $$CURRENT_VERSION, applying aggregator migrations..."; \
			TEMP_MIGRATIONS=$$(mktemp -d); \
			if [ -d "../rule-service/migrations" ]; then cp -r ../rule-service/migrations/* $$TEMP_MIGRATIONS/ 2>/dev/null || true; fi; \
			cp -r migrations/* $$TEMP_MIGRATIONS/ 2>/dev/null || true; \
			$$MIGRATE -path $$TEMP_MIGRATIONS -database "$$POSTGRES_DSN" up; \
			rm -rf $$TEMP_MIGRATIONS; \
		fi; \
	else \
		echo "Warning: Could not find Postgres container, using aggregator migrations only..."; \
		$$MIGRATE -path migrations -database "$$POSTGRES_DSN" up; \
	fi

migrate-down:
	@echo "Running database migrations (down)..."
	@MIGRATE=$$(command -v migrate 2>/dev/null || [ -f ~/go/bin/migrate ] && echo ~/go/bin/migrate || [ -n "$$GOPATH" ] && [ -f $$GOPATH/bin/migrate ] && echo $$GOPATH/bin/migrate || echo ""); \
	if [ -z "$$MIGRATE" ]; then \
		echo "Error: migrate tool not found."; \
		echo "Install with: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"; \
		echo "If installed, ensure ~/go/bin is in your PATH or set GOPATH/bin in PATH"; \
		exit 1; \
	fi; \
	$$MIGRATE -path migrations -database "$${POSTGRES_DSN:-postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable}" down

migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make migrate-create NAME=migration_name"; \
		exit 1; \
	fi
	@MIGRATE=$$(command -v migrate 2>/dev/null || [ -f ~/go/bin/migrate ] && echo ~/go/bin/migrate || [ -n "$$GOPATH" ] && [ -f $$GOPATH/bin/migrate ] && echo $$GOPATH/bin/migrate || echo ""); \
	if [ -z "$$MIGRATE" ]; then \
		echo "Error: migrate tool not found."; \
		echo "Install with: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"; \
		echo "If installed, ensure ~/go/bin is in your PATH or set GOPATH/bin in PATH"; \
		exit 1; \
	fi; \
	$$MIGRATE create -ext sql -dir migrations -seq $(NAME)

# Database query targets (uses centralized infrastructure)
# Note: Infrastructure is managed at root level - use 'make setup-infra' from root
db-query:
	@echo "Querying notifications table..."
	@docker exec -it alerting-platform-postgres psql -U postgres -d alerting -c "SELECT notification_id, client_id, alert_id, severity, source, name, status, created_at FROM notifications ORDER BY created_at DESC LIMIT 20;"

db-count:
	@echo "Counting notifications..."
	@docker exec -it alerting-platform-postgres psql -U postgres -d alerting -c "SELECT COUNT(*) as total_notifications, status, COUNT(*) FILTER (WHERE status = 'RECEIVED') as received, COUNT(*) FILTER (WHERE status = 'SENT') as sent FROM notifications GROUP BY status;"

db-all:
	@echo "All notifications:"
	@docker exec -it alerting-platform-postgres psql -U postgres -d alerting -c "SELECT * FROM notifications ORDER BY created_at DESC;"

db-psql:
	@echo "Opening PostgreSQL interactive shell..."
	@docker exec -it alerting-platform-postgres psql -U postgres -d alerting

db-received:
	@echo "Notifications with status RECEIVED:"
	@docker exec -it alerting-platform-postgres psql -U postgres -d alerting -c "SELECT notification_id, client_id, alert_id, severity, source, name, created_at FROM notifications WHERE status = 'RECEIVED' ORDER BY created_at DESC;"

# Run everything: verify dependencies, setup, and start aggregator
run-all:
	@chmod +x scripts/run-all.sh
	@./scripts/run-all.sh
