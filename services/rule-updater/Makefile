.PHONY: build run test clean help deps run-all

BINARY_NAME=rule-updater
CMD_PATH=./cmd/rule-updater

help:
	@echo "Available targets:"
	@echo "  run-all       - Verify dependencies and run rule-updater (recommended)"
	@echo "  build         - Build the binary"
	@echo "  run           - Run the service"
	@echo "  test          - Run tests"
	@echo "  clean         - Remove build artifacts"
	@echo "  deps          - Download dependencies"
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

# Run everything: verify dependencies and start rule-updater
run-all:
	@chmod +x scripts/run-all.sh
	@./scripts/run-all.sh
