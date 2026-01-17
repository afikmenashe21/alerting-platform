.PHONY: build run test clean help deps

BINARY_NAME=rule-updater
CMD_PATH=./cmd/rule-updater

help:
	@echo "Available targets:"
	@echo "  run-all       - Verify dependencies, setup, and run rule-updater (recommended)"
	@echo "  build         - Build the binary"
	@echo "  run           - Run the service"
	@echo "  test          - Run tests"
	@echo "  clean         - Remove build artifacts"
	@echo "  deps          - Download dependencies"
	@echo ""
	@echo "Docker/Kafka/Postgres/Redis targets:"
	@echo "  up            - Start Kafka, Postgres, Redis, and dependencies"
	@echo "  down          - Stop services"
	@echo "  logs          - Show service logs"
	@echo "  status        - Show service status"
	@echo "  create-topics - Create Kafka topics"

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

# Docker/Kafka/Postgres/Redis targets
up:
	@echo "Starting Kafka, Postgres, Redis, and dependencies..."
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
	@docker exec rule-updater-kafka kafka-topics --create \
		--bootstrap-server localhost:9092 \
		--topic rule.changed \
		--partitions 3 \
		--replication-factor 1 \
		--if-not-exists 2>/dev/null || echo "Topic rule.changed already exists or creation failed"
	@echo "Topics created/verified"

# Setup: start services, create topics
setup: up
	@echo "Waiting for services to be ready..."
	@sleep 5
	@echo "Creating Kafka topics..."
	@$(MAKE) create-topics
	@echo "Setup complete!"

# Run everything: verify dependencies, setup, and start rule-updater
run-all:
	@chmod +x scripts/run-all.sh
	@./scripts/run-all.sh
