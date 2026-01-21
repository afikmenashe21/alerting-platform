.PHONY: build run test clean help deps run-all db-query db-count db-sent db-psql

BINARY_NAME=sender
CMD_PATH=./cmd/sender

help:
	@echo "Available targets:"
	@echo "  run-all       - Verify dependencies and run sender (recommended)"
	@echo "  build         - Build the binary"
	@echo "  run           - Run the service"
	@echo "  test          - Run tests"
	@echo "  clean         - Remove build artifacts"
	@echo "  deps          - Download dependencies"
	@echo ""
	@echo "Database targets (uses centralized infrastructure):"
	@echo "  db-query      - Show recent notifications (last 20)"
	@echo "  db-count      - Show notification counts by status"
	@echo "  db-sent       - Show notifications with status SENT"
	@echo "  db-psql       - Open PostgreSQL interactive shell"
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

# Database query targets (uses centralized infrastructure)
# Note: Infrastructure is managed at root level - use 'make setup-infra' from root
db-query:
	@echo "Querying notifications table..."
	@docker exec -it alerting-platform-postgres psql -U postgres -d alerting -c "SELECT notification_id, client_id, alert_id, severity, source, name, status, created_at FROM notifications ORDER BY created_at DESC LIMIT 20;"

db-count:
	@echo "Counting notifications..."
	@docker exec -it alerting-platform-postgres psql -U postgres -d alerting -c "SELECT COUNT(*) as total_notifications, status, COUNT(*) FILTER (WHERE status = 'RECEIVED') as received, COUNT(*) FILTER (WHERE status = 'SENT') as sent FROM notifications GROUP BY status;"

db-sent:
	@echo "Notifications with status SENT:"
	@docker exec -it alerting-platform-postgres psql -U postgres -d alerting -c "SELECT notification_id, client_id, alert_id, severity, source, name, updated_at FROM notifications WHERE status = 'SENT' ORDER BY updated_at DESC;"

db-psql:
	@echo "Opening PostgreSQL interactive shell..."
	@docker exec -it alerting-platform-postgres psql -U postgres -d alerting

# Run everything: verify dependencies and start sender
run-all:
	@chmod +x scripts/run-all.sh
	@./scripts/run-all.sh
