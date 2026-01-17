.PHONY: build run run-test test clean help

BINARY_NAME=alert-producer
CMD_PATH=./cmd/alert-producer

help:
	@echo "Available targets:"
	@echo "  setup-run    - Complete setup and run (recommended)"
	@echo "  build        - Build the binary"
	@echo "  run          - Run the service (default: 10 RPS for 60s)"
	@echo "  run-test     - Run in test mode (generates LOW/test-source/test-name alerts)"
	@echo "  run-single-test - Send a single test alert (LOW/test-source/test-name) and exit"
	@echo "  test         - Run tests"
	@echo "  clean        - Remove build artifacts"
	@echo "  deps         - Download dependencies"

build:
	@echo "Building $(BINARY_NAME)..."
	go build -o bin/$(BINARY_NAME) $(CMD_PATH)

run: build
	@echo "Running $(BINARY_NAME)..."
	./bin/$(BINARY_NAME) $(ARGS)

run-test: build
	@echo "Running $(BINARY_NAME) in test mode (LOW/test-source/test-name)..."
	./bin/$(BINARY_NAME) -test $(ARGS)

run-single-test: build
	@echo "Running $(BINARY_NAME) in single test mode (sending one alert: LOW/test-source/test-name)..."
	./bin/$(BINARY_NAME) -single-test $(ARGS)

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

# Example usage targets
run-burst:
	$(MAKE) run ARGS="-burst 100 -kafka-brokers localhost:9092"

run-continuous:
	$(MAKE) run ARGS="-rps 50 -duration 5m -kafka-brokers localhost:9092"

run-deterministic:
	$(MAKE) run ARGS="-rps 10 -duration 30s -seed 42 -kafka-brokers localhost:9092"

# Test mode examples
run-test-burst:
	$(MAKE) run-test ARGS="-burst 10 -kafka-brokers localhost:9092"

run-test-continuous:
	$(MAKE) run-test ARGS="-rps 5 -duration 30s -kafka-brokers localhost:9092"

# Docker/Kafka targets
# Note: Uses 'docker compose' (V2) - if you have V1, use 'docker-compose' instead
kafka-up:
	@echo "Starting Kafka..."
	@docker compose up -d 2>/dev/null || docker-compose up -d
	@echo "Waiting for Kafka to be ready..."
	@sleep 10
	@echo "Kafka should be ready. Check with: docker compose ps"

kafka-down:
	@echo "Stopping Kafka..."
	@docker compose down 2>/dev/null || docker-compose down

kafka-logs:
	@docker compose logs -f kafka 2>/dev/null || docker-compose logs -f kafka

kafka-status:
	@docker compose ps 2>/dev/null || docker-compose ps

kafka-create-topic:
	@echo "Creating topic alerts.new..."
	@docker exec kafka kafka-topics --create \
		--bootstrap-server localhost:9092 \
		--topic alerts.new \
		--partitions 3 \
		--replication-factor 1 \
		--if-not-exists 2>/dev/null || echo "Topic creation failed or already exists"

# Complete setup and run (recommended)
# Note: Infrastructure should be started centrally with: cd ../.. && make setup-infra
setup-run:
	@echo "Running service (infrastructure should be started separately)..."
	@./scripts/run-all.sh $(ARGS)

# Test targets
test-integration:
	@echo "Running integration tests..."
	./scripts/test-producer.sh
