.PHONY: build run test clean help deps

BINARY_NAME=evaluator
CMD_PATH=./cmd/evaluator

help:
	@echo "Available targets:"
	@echo "  run-all  - Verify dependencies, setup, and run evaluator (recommended)"
	@echo "  build    - Build the binary"
	@echo "  run      - Run the service"
	@echo "  test     - Run tests"
	@echo "  clean    - Remove build artifacts"
	@echo "  deps     - Download dependencies"

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

# Example usage targets
run-default:
	$(MAKE) run ARGS="-kafka-brokers localhost:9092 -redis-addr localhost:6379"

# Create test snapshot in Redis (for testing before rule-updater is ready)
create-test-snapshot:
	@echo "Creating test rule snapshot in Redis..."
	@cd scripts && go run create-test-snapshot.go localhost:6379

# Run everything: verify dependencies, setup, and start evaluator
run-all:
	@chmod +x scripts/run-all.sh
	@./scripts/run-all.sh

# Note: Infrastructure (Kafka, Redis, Postgres) is managed centrally
# Start infrastructure from root: cd ../.. && make setup-infra
