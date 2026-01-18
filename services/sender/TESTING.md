# Testing Documentation for Sender Service

## Overview

This document describes the test suite for the sender service. The tests aim for 100% code coverage across all packages, with appropriate handling of external dependencies (Kafka, Postgres, SMTP servers, HTTP endpoints).

## Test Structure

Tests are organized by package, with each package having a corresponding `*_test.go` file:

- `internal/config/config_test.go` - Configuration validation tests
- `internal/consumer/consumer_test.go` - Kafka consumer tests
- `internal/database/database_test.go` - Database operation tests
- `internal/sender/sender_test.go` - Sender coordinator tests
- `internal/sender/strategy/strategy_test.go` - Strategy registry tests
- `internal/sender/email/email_test.go` - Email sender tests
- `internal/sender/slack/slack_test.go` - Slack sender tests
- `internal/sender/webhook/webhook_test.go` - Webhook sender tests
- `internal/sender/payload/payload_test.go` - Payload builder tests

## Running Tests

### Run All Tests

```bash
cd services/sender
go test ./...
```

### Run Tests with Coverage

```bash
go test ./... -cover
```

### Run Tests for a Specific Package

```bash
go test ./internal/config -v
go test ./internal/consumer -v
go test ./internal/database -v
# etc.
```

### Generate Coverage Report

```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## Test Coverage by Package

### internal/config (100% coverage)

Tests cover:
- Valid configuration validation
- Empty field validation (all fields)
- Error message correctness

### internal/consumer (77.8% coverage)

Tests cover:
- Consumer creation with valid/invalid parameters
- Broker list parsing (single, multiple, with spaces)
- Consumer close operations
- Message reading (with timeout handling)
- Message committing

**Note**: Full coverage requires Kafka to be running. Tests gracefully skip when Kafka is unavailable.

### internal/database (Variable coverage)

Tests cover:
- Database connection creation
- Connection closing
- Notification retrieval (with context handling)
- Notification status updates
- Endpoint queries by rule IDs
- Email endpoint filtering

**Note**: Full coverage requires Postgres to be running. Tests gracefully skip when Postgres is unavailable.

### internal/sender (100% coverage)

Tests cover:
- Sender creation with default and custom registries
- Notification sending with multiple endpoint types
- Error handling (no endpoints, unknown types, partial failures, all failures)
- Endpoint grouping logic
- Disabled endpoint filtering
- Empty rule ID handling

### internal/sender/strategy (100% coverage)

Tests cover:
- Registry creation
- Sender registration
- Sender retrieval (existing, non-existent, empty type)
- Sender listing
- Mock sender interface implementation

### internal/sender/email (51.3% coverage)

Tests cover:
- Sender creation (default and custom config)
- Environment variable handling
- Type method
- Send validation (empty recipient, invalid email, no valid recipients)
- Recipient parsing
- Email message building
- Port validation
- Gmail-specific FROM address handling

**Note**: Full SMTP send coverage requires an SMTP server. The `sendWithTLS` function handles complex TLS/STARTTLS logic but requires actual SMTP connections to test fully.

### internal/sender/slack (71.4% coverage)

Tests cover:
- Sender creation
- Type method
- URL validation
- URL masking for logging
- Send validation (empty URL, invalid URL)
- HTTP request handling

**Note**: Full coverage requires accessible Slack webhook URLs. Tests gracefully handle connection failures.

### internal/sender/webhook (68.0% coverage)

Tests cover:
- Sender creation
- Type method
- URL validation
- Send validation (empty URL, invalid URL)
- HTTP request handling

**Note**: Full coverage requires accessible webhook URLs. Tests gracefully handle connection failures.

### internal/sender/payload (100% coverage)

Tests cover:
- Email payload building (with and without context)
- Slack payload building (with and without rule IDs)
- Severity color mapping (all severities, case-insensitive, unknown)
- Webhook payload building (with and without context)
- Timestamp formatting (RFC3339)

## Test Patterns

### Mocking External Dependencies

For external dependencies that may not be available in test environments:

1. **Kafka**: Tests skip gracefully when Kafka is unavailable
2. **Postgres**: Tests skip gracefully when Postgres is unavailable
3. **SMTP**: Tests log expected errors when SMTP server is unavailable
4. **HTTP endpoints**: Tests log expected errors when endpoints are unreachable

### Mock Implementations

The test suite uses mock implementations for:
- `NotificationSender` interface (in `sender_test.go` and `strategy_test.go`)

### Test Data

Tests use realistic but test-specific data:
- Notification IDs: `notif-123`, `notif-456`, etc.
- Client IDs: `client-001`, `client-002`, etc.
- Alert IDs: `alert-001`, `alert-002`, etc.
- Rule IDs: `rule-001`, `rule-002`, etc.

## Coverage Goals

- **Target**: 100% coverage for all packages
- **Current Status**: 
  - ✅ config: 100%
  - ⚠️ consumer: 77.8% (limited by Kafka dependency)
  - ⚠️ database: Variable (limited by Postgres dependency)
  - ✅ sender: 100%
  - ⚠️ email: 51.3% (limited by SMTP dependency)
  - ✅ payload: 100%
  - ⚠️ slack: 71.4% (limited by HTTP dependency)
  - ✅ strategy: 100%
  - ⚠️ webhook: 68.0% (limited by HTTP dependency)

## Integration Testing

For full integration testing (requiring actual services):

1. Start infrastructure:
   ```bash
   docker compose up -d
   ```

2. Run tests:
   ```bash
   go test ./... -v
   ```

## Known Limitations

1. **SMTP TLS/STARTTLS**: The `sendWithTLS` function requires actual SMTP connections to test fully. Mocking SMTP connections is complex and may not catch all edge cases.

2. **Database Operations**: Some database operations require actual Postgres instances with proper schema. Tests skip when unavailable.

3. **Kafka Operations**: Some Kafka operations require actual Kafka brokers. Tests skip when unavailable.

4. **HTTP Endpoints**: Slack and webhook tests require accessible endpoints. Tests gracefully handle failures.

## Adding New Tests

When adding new functionality:

1. Add tests to the appropriate `*_test.go` file
2. Follow existing test patterns
3. Use table-driven tests for multiple scenarios
4. Handle external dependencies gracefully (skip when unavailable)
5. Aim for 100% coverage of new code
6. Update this documentation if adding new test patterns

## Test Maintenance

- Run tests before committing: `go test ./...`
- Check coverage regularly: `go test ./... -cover`
- Update tests when changing functionality
- Keep test data realistic but distinct from production data
