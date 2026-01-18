# Test Organization Strategy

## Current Structure (Go Standard Convention)

Tests are placed alongside source files using the `*_test.go` naming convention:

```
internal/
├── config/
│   ├── config.go
│   └── config_test.go
├── consumer/
│   ├── consumer.go
│   └── consumer_test.go
├── database/
│   ├── database.go
│   └── database_test.go
├── producer/
│   ├── producer.go
│   └── producer_test.go
└── processor/
    ├── processor.go
    └── processor_test.go
```

## Why This Structure?

### ✅ Advantages

1. **Co-location**: Tests are next to the code they test, making it easy to find and maintain
2. **Package Access**: Tests in the same package can access unexported (private) functions and types
3. **Go Standard**: This is the idiomatic Go way, matching the existing codebase pattern
4. **Tooling Support**: `go test ./...` automatically discovers and runs all tests
5. **IDE Support**: Most Go IDEs expect this structure

### ⚠️ Alternative: Separate Test Directory

Some projects use a separate `test/` or `tests/` directory:

```
internal/
├── config/
│   └── config.go
test/
├── config/
│   └── config_test.go
```

**Trade-offs:**
- ✅ Clear separation of tests from source
- ❌ Tests must use exported APIs only (can't test private functions)
- ❌ Less common in Go ecosystem
- ❌ Requires different package names (`package config_test` instead of `package config`)

## Hybrid Approach (Recommended for Large Projects)

Use both structures:

```
internal/
├── config/
│   ├── config.go
│   └── config_test.go          ← Unit tests (same package)
test/
├── integration/
│   └── aggregator_integration_test.go  ← Integration tests
└── fixtures/
    └── test_data.go            ← Test utilities
```

## Current Project Pattern

The existing codebase (alert-producer, etc.) follows the standard Go convention:
- `*_test.go` files in the same package directory
- Integration test scripts in `scripts/test/` directory
- This aggregator service follows the same pattern for consistency

## Recommendation

**Keep the current structure** because:
1. It matches the existing codebase pattern
2. It's the Go standard convention
3. It allows testing private functions when needed
4. It's easier to maintain (tests next to code)

If you prefer a separate test directory, we can reorganize, but you'll lose the ability to test private functions directly.
