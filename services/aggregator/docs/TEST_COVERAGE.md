# Aggregator Service Test Coverage

## Test Files Created

1. **internal/config/config_test.go** - ✅ Complete
   - Tests `Config.Validate()` with all validation scenarios
   - 100% coverage of config package

2. **internal/database/database_test.go** - ⚠️ Partial
   - Tests `NewDB()` with invalid DSN
   - Tests `Close()` with nil connection
   - `InsertNotificationIdempotent()` requires sqlmock for full coverage
   - **To achieve 100% coverage:**
     - Install sqlmock: `go get github.com/DATA-DOG/go-sqlmock`
     - Add tests for all InsertNotificationIdempotent scenarios (see comments in test file)

3. **internal/consumer/consumer_test.go** - ⚠️ Partial
   - Tests `NewConsumer()` validation logic
   - `ReadMessage()`, `CommitMessage()`, and `Close()` require real Kafka or interface refactoring
   - **To achieve 100% coverage:**
     - Refactor to use interfaces for dependency injection, OR
     - Use integration tests with testcontainers or real Kafka instance

4. **internal/producer/producer_test.go** - ⚠️ Partial
   - Tests `NewProducer()` validation logic
   - `Publish()` and `Close()` require real Kafka or interface refactoring
   - **To achieve 100% coverage:**
     - Refactor to use interfaces for dependency injection, OR
     - Use integration tests with testcontainers or real Kafka instance

5. **internal/processor/processor_test.go** - ⚠️ Partial
   - Tests `NewProcessor()` constructor
   - `ProcessNotifications()` requires mocks for consumer, producer, and db
   - **To achieve 100% coverage:**
     - Refactor consumer, producer, and database to use interfaces
     - Create mocks for all dependencies
     - Test all code paths in ProcessNotifications

## Current Coverage Status

- **config**: ~100% (all testable code covered)
- **database**: ~40% (validation covered, needs sqlmock for full coverage)
- **consumer**: ~30% (validation covered, needs Kafka mocks/interfaces)
- **producer**: ~30% (validation covered, needs Kafka mocks/interfaces)
- **processor**: ~20% (constructor covered, needs full refactoring for ProcessNotifications)

## To Achieve 100% Coverage

### Option 1: Interface-Based Refactoring (Recommended)

Refactor the service to use interfaces:

1. **Consumer Interface:**
   ```go
   type AlertConsumer interface {
       ReadMessage(ctx context.Context) (*events.AlertMatched, *kafka.Message, error)
       CommitMessage(ctx context.Context, msg *kafka.Message) error
       Close() error
   }
   ```

2. **Producer Interface:**
   ```go
   type NotificationProducer interface {
       Publish(ctx context.Context, ready *events.NotificationReady) error
       Close() error
   }
   ```

3. **Database Interface:**
   ```go
   type NotificationDB interface {
       InsertNotificationIdempotent(ctx context.Context, ...) (*string, error)
       Close() error
   }
   ```

4. Update `Processor` to accept interfaces instead of concrete types
5. Create mock implementations for testing
6. Write comprehensive tests for all code paths

### Option 2: Integration Tests

Use testcontainers or real infrastructure:
- Set up test Kafka instance
- Set up test PostgreSQL database
- Write integration tests that exercise all code paths

### Option 3: Hybrid Approach

- Use interfaces for unit testing critical logic
- Use integration tests for end-to-end validation

## Running Tests

```bash
# Run all tests
go test ./... -v

# Run with coverage
go test ./... -cover

# Run with detailed coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Dependencies Needed

For full test coverage, install:
```bash
go get github.com/DATA-DOG/go-sqlmock
```

## Notes

- The current test structure focuses on what can be tested without major refactoring
- Validation logic is fully covered
- Integration with Kafka and database requires either:
  - Interface refactoring (better for unit tests)
  - Integration tests (better for end-to-end validation)
- The processor's `ProcessNotifications` method has complex logic that should be thoroughly tested with all error paths and edge cases
