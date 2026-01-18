# Test Coverage Documentation

## Overview

This document describes the test coverage for the `alert-producer` service. The test suite aims for comprehensive coverage of all functionality, with special attention to error handling, edge cases, and integration scenarios.

## Current Coverage

As of the latest test run, the overall test coverage is **87.4%**:

- **config**: 97.4% coverage
- **generator**: 93.3% coverage
- **processor**: 90.8% coverage
- **producer**: 67.3% coverage

## Test Organization

### Package Structure

Tests are organized alongside the source code in `*_test.go` files:

```
internal/
├── config/
│   ├── config.go
│   └── config_test.go
├── generator/
│   ├── generator.go
│   └── generator_test.go
├── processor/
│   ├── processor.go
│   └── processor_test.go
└── producer/
    ├── producer.go
    ├── mock_producer.go
    └── producer_test.go
```

## Test Coverage by Package

### config Package (97.4%)

**Tested:**
- ✅ `ParseDistribution` - All valid and invalid input formats
- ✅ `Config.Validate` - All validation scenarios including edge cases
- ✅ Empty strings, invalid formats, percentage validation
- ✅ Whitespace handling, multiple values, single values

**Coverage Gaps:**
- Some extremely rare edge cases in distribution parsing (empty parts handling)

### generator Package (93.3%)

**Tested:**
- ✅ `New` - With and without seed, deterministic behavior
- ✅ `Generate` - Alert generation with all required fields
- ✅ `GenerateBoilerplate` - Fixed-value alert generation
- ✅ `GenerateTestAlert` - Test alert generation
- ✅ `selectWeighted` - Weighted selection with various distributions
- ✅ `selectFrom` - Uniform selection from choices
- ✅ Empty choices handling
- ✅ Context field generation (probabilistic)

**Coverage Gaps:**
- Panic path in `New` when invalid distribution is passed (defensive code)
- Fallback path in `selectWeighted` (extremely unlikely edge case)

### processor Package (90.8%)

**Tested:**
- ✅ `NewProcessor` - Processor initialization
- ✅ `Process` - Main processing with burst and continuous modes
- ✅ `ProcessBurst` - Burst mode processing
- ✅ `ProcessContinuous` - Continuous mode processing
- ✅ `ProcessTest` - Test mode with varied alerts
- ✅ Context cancellation handling
- ✅ Error handling for publish failures
- ✅ Progress logging
- ✅ All internal methods (runBurstMode, runContinuousMode, etc.)

**Coverage Gaps:**
- Some edge cases in continuous mode timing (duration expiration vs. ticker)
- Very specific timing-dependent scenarios

### producer Package (67.3%)

**Tested:**
- ✅ `New` - Producer initialization with valid/invalid inputs
- ✅ `Publish` - Message publishing (when Kafka available)
- ✅ `Close` - Graceful shutdown
- ✅ `hashAlertID` - Hash function with various inputs
- ✅ `NewMock` - Mock producer creation
- ✅ `MockProducer.Publish` - Mock publishing
- ✅ `MockProducer.Close` - Mock cleanup
- ✅ Context timeout handling
- ✅ Error handling for closed producer

**Coverage Gaps:**
- `createTopicIfNotExists` - Requires Kafka infrastructure (26.7% coverage)
  - Topic creation success path
  - Topic already exists path
  - Connection failure path (partially tested)
- `Publish` error paths that require Kafka connection failures (70% coverage)
- `Close` error path when writer.Close() fails (66.7% coverage)
- `MockProducer.Publish` error path for JSON marshaling failures (66.7% coverage)

**Note:** The producer package has lower coverage because many code paths require actual Kafka infrastructure or complex mocking of the `kafka-go` library. The tests are designed to work both with and without Kafka, gracefully skipping integration tests when Kafka is unavailable.

## Running Tests

### Run All Tests

```bash
go test ./internal/... -v
```

### Run Tests with Coverage

```bash
go test ./internal/... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

### Generate HTML Coverage Report

```bash
go test ./internal/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Run Tests for Specific Package

```bash
go test ./internal/config -v
go test ./internal/generator -v
go test ./internal/processor -v
go test ./internal/producer -v
```

## Test Categories

### Unit Tests

- **config**: Pure unit tests, no dependencies
- **generator**: Pure unit tests with deterministic RNG
- **processor**: Unit tests with mock publisher
- **producer**: Unit tests for hash function and mock producer

### Integration Tests

- **producer**: Tests that require Kafka (skipped if Kafka unavailable)
  - Real Kafka producer creation
  - Message publishing to Kafka
  - Topic creation

### Mock Tests

- **processor**: Uses mock publisher to test processing logic without Kafka
- **producer**: MockProducer tests for testing without Kafka

## Test Patterns

### Mock Publisher

The processor tests use a `mockPublisher` that implements the `AlertPublisher` interface:

```go
type mockPublisher struct {
    published []*generator.Alert
    shouldErr bool
    errMsg    string
    closed    bool
}
```

This allows testing processor logic without requiring Kafka.

### Deterministic Testing

Generator tests use fixed seeds to ensure deterministic behavior:

```go
cfg := config.Config{
    Seed: 42, // Fixed seed for reproducibility
}
```

### Context Cancellation

Many tests verify graceful handling of context cancellation:

```go
ctx, cancel := context.WithCancel(context.Background())
cancel() // Cancel immediately
err := proc.ProcessBurst(ctx, 100)
// Verify error handling
```

### Error Path Testing

Tests verify error handling for:
- Invalid configuration
- Publish failures
- Context timeouts
- Empty/invalid inputs

## Known Limitations

### Infrastructure Dependencies

Some code paths require external infrastructure:
- Kafka connection for producer integration tests
- Topic creation requires Kafka admin API access

These tests gracefully skip when infrastructure is unavailable.

### Timing-Dependent Tests

Some processor tests use timing (RPS, duration) which can be flaky:
- Tests use higher RPS (100.0) and longer durations (200ms) to reduce flakiness
- Continuous mode tests verify behavior over time windows

### Hard-to-Test Paths

Some defensive code paths are difficult to test:
- Panic handlers in generator initialization
- Fallback paths that should never execute
- Error paths in Kafka library that require specific failure conditions

## Improving Coverage

To improve test coverage further:

1. **Mock Kafka Library**: Use a mocking library for `kafka-go` to test error paths
2. **Test Containers**: Use testcontainers to spin up real Kafka instances for integration tests
3. **Error Injection**: Add error injection points for testing error handling
4. **Edge Case Expansion**: Add more edge case tests for boundary conditions

## Test Maintenance

### Adding New Tests

When adding new functionality:

1. Add tests alongside the code in `*_test.go` files
2. Follow existing test patterns (table-driven tests, mock usage)
3. Ensure tests pass both with and without Kafka
4. Update this documentation if coverage significantly changes

### Test Naming

Follow Go conventions:
- Test functions: `TestFunctionName` or `TestStruct_MethodName`
- Subtests: Use `t.Run("subtest name", ...)`
- Test files: `*_test.go` in the same package

### Test Data

- Use deterministic seeds for reproducible tests
- Use helper functions for common test data creation
- Keep test data minimal and focused

## Continuous Integration

Tests should be run:
- Before every commit
- In CI/CD pipelines
- Before releases
- When adding new features

## Summary

The test suite provides comprehensive coverage of the alert-producer service functionality. The remaining uncovered code is primarily:
1. Infrastructure-dependent code (Kafka integration)
2. Defensive programming paths (panic handlers, fallbacks)
3. Extremely rare edge cases

The tests are designed to be maintainable, fast, and reliable, with clear separation between unit tests (no dependencies) and integration tests (require infrastructure).
