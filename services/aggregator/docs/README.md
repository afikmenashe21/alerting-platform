# Aggregator Service Documentation

This directory contains all documentation for the aggregator service.

## 📖 Documentation Index

### Testing Documentation
- **[Test Coverage](./TEST_COVERAGE.md)** - Test coverage status and requirements for 100% coverage
- **[Test Organization](./TEST_ORGANIZATION.md)** - Test structure and organization strategy
- **[Test Results](./TEST_RESULTS.md)** - Latest test execution results and status

## 🚀 Quick Links

- **Main README**: [../README.md](../README.md)
- **Makefile**: [../Makefile](../Makefile) - Build and run commands
- **Docker Compose**: [../../docker-compose.yml](../../docker-compose.yml) - Centralized infrastructure setup

## Service Overview

The aggregator service:
- Consumes matched alerts from `alerts.matched` topic
- Performs idempotent deduplication using database unique constraints
- Publishes notification ready events to `notifications.ready` topic
- Ensures at-least-once delivery semantics

## Running Tests

```bash
# Run all tests
go test ./... -v

# Run with coverage
go test ./... -cover

# Run specific package
go test ./internal/config -v
```
