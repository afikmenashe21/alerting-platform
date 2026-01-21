.PHONY: build run test clean help deps migrate-up migrate-down migrate-create run-all db-query db-clients db-rules db-psql

BINARY_NAME=rule-service
CMD_PATH=./cmd/rule-service

help:
	@echo "Available targets:"
	@echo "  run-all       - Verify dependencies and run (RECOMMENDED)"
	@echo "  build         - Build the binary"
	@echo "  run           - Run the service"
	@echo "  test          - Run tests"
	@echo "  clean         - Remove build artifacts"
	@echo "  deps          - Download dependencies"
	@echo ""
	@echo "Database targets (uses centralized infrastructure):"
	@echo "  db-query      - Show recent rules (last 20)"
	@echo "  db-clients    - Show all clients"
	@echo "  db-rules      - Show all rules"
	@echo "  db-psql       - Open PostgreSQL interactive shell"
	@echo ""
	@echo "Migration targets:"
	@echo "  migrate-up    - Run database migrations (up)"
	@echo "  migrate-down  - Run database migrations (down)"
	@echo "  migrate-create - Create a new migration (usage: make migrate-create NAME=migration_name)"
	@echo ""
	@echo "Note: Infrastructure is managed at root level. Use 'make setup-infra' from root."

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
	$$MIGRATE -path migrations -database "$${POSTGRES_DSN:-postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable}" up

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
	@echo "Querying rules table..."
	@docker exec -it alerting-platform-postgres psql -U postgres -d alerting -c "SELECT rule_id, client_id, severity, source, name, enabled, version, updated_at FROM rules ORDER BY updated_at DESC LIMIT 20;"

db-clients:
	@echo "All clients:"
	@docker exec -it alerting-platform-postgres psql -U postgres -d alerting -c "SELECT * FROM clients ORDER BY created_at DESC;"

db-rules:
	@echo "All rules:"
	@docker exec -it alerting-platform-postgres psql -U postgres -d alerting -c "SELECT * FROM rules ORDER BY created_at DESC;"

db-psql:
	@echo "Opening PostgreSQL interactive shell..."
	@docker exec -it alerting-platform-postgres psql -U postgres -d alerting

# Run everything: verify dependencies and start rule-service
run-all:
	@chmod +x scripts/run-all.sh 2>/dev/null || true
	@./scripts/run-all.sh

# Test rule.changed event publishing
test-events:
	@chmod +x scripts/test-rule-events.sh 2>/dev/null || true
	@./scripts/test-rule-events.sh
