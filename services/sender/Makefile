.PHONY: build run test clean help deps run-all

BINARY_NAME=sender
CMD_PATH=./cmd/sender

help:
	@echo "Available targets:"
	@echo "  run-all       - Verify dependencies, setup, and run sender (recommended)"
	@echo "  build         - Build the binary"
	@echo "  run           - Run the service"
	@echo "  test          - Run tests"
	@echo "  clean         - Remove build artifacts"
	@echo "  deps          - Download dependencies"
	@echo ""
	@echo "Database targets:"
	@echo "  db-query      - Show recent notifications (last 20)"
	@echo "  db-count      - Show notification counts by status"
	@echo "  db-sent       - Show notifications with status SENT"
	@echo "  db-psql       - Open PostgreSQL interactive shell"

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

# Docker/Kafka/Postgres targets
up:
	@echo "Starting Kafka, Postgres, and dependencies..."
	@docker compose up -d 2>/dev/null || docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 10
	@echo "Services should be ready. Check with: docker compose ps"

down:
	@echo "Stopping services..."
	@docker compose down 2>/dev/null || docker-compose down

logs:
	@docker compose logs -f 2>/dev/null || docker-compose logs -f

status:
	@docker compose ps 2>/dev/null || docker-compose ps

create-topics:
	@echo "Creating Kafka topics..."
	@docker exec sender-kafka kafka-topics --create \
		--bootstrap-server localhost:9092 \
		--topic notifications.ready \
		--partitions 3 \
		--replication-factor 1 \
		--if-not-exists 2>/dev/null || echo "Topic notifications.ready already exists or creation failed"
	@echo "Topics created/verified"

# Setup: start services, create topics
setup: up
	@echo "Waiting for Postgres to be ready..."
	@sleep 5
	@echo "Creating Kafka topics..."
	@$(MAKE) create-topics
	@echo "Setup complete!"

# Database query targets
db-query:
	@echo "Querying notifications table..."
	@docker exec -it sender-postgres psql -U postgres -d alerting -c "SELECT notification_id, client_id, alert_id, severity, source, name, status, created_at FROM notifications ORDER BY created_at DESC LIMIT 20;"

db-count:
	@echo "Counting notifications..."
	@docker exec -it sender-postgres psql -U postgres -d alerting -c "SELECT COUNT(*) as total_notifications, status, COUNT(*) FILTER (WHERE status = 'RECEIVED') as received, COUNT(*) FILTER (WHERE status = 'SENT') as sent FROM notifications GROUP BY status;"

db-sent:
	@echo "Notifications with status SENT:"
	@docker exec -it sender-postgres psql -U postgres -d alerting -c "SELECT notification_id, client_id, alert_id, severity, source, name, updated_at FROM notifications WHERE status = 'SENT' ORDER BY updated_at DESC;"

db-psql:
	@echo "Opening PostgreSQL interactive shell..."
	@docker exec -it sender-postgres psql -U postgres -d alerting

# Run everything: verify dependencies, setup, and start sender
run-all:
	@chmod +x scripts/run-all.sh
	@./scripts/run-all.sh
