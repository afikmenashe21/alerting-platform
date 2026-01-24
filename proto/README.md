# Protobuf Definitions

This directory contains Protocol Buffer (protobuf) definitions for all Kafka event types used in the alerting platform.

## Files

- `common.proto` - Shared enums (Severity, RuleAction)
- `alerts.proto` - AlertNew and AlertMatched messages
- `rules.proto` - RuleChanged message
- `notifications.proto` - NotificationReady message

## Prerequisites

Before generating Go code, you need:

1. **protoc** (Protocol Buffer Compiler)
   - macOS: `brew install protobuf`
   - Linux: `sudo apt-get install protobuf-compiler` (Ubuntu/Debian)
   - Or download from: https://grpc.io/docs/protoc-installation/

2. **protoc-gen-go** (Go plugin)
   ```bash
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   ```

3. **buf** (recommended, for linting and breaking change detection)
   - macOS: `brew install bufbuild/buf/buf`
   - Linux: See https://buf.build/docs/installation
   - Provides advanced linting and schema evolution checks

## Generating Go Code

From the repository root:

```bash
# Install dependencies (first time only)
make proto-install-deps

# Generate Go code
make proto-generate

# Validate proto files
make proto-validate

# Verify generated code is up-to-date
make proto-verify-generated

# Lint proto files (requires buf)
make proto-lint

# Check for breaking changes (requires buf)
make proto-breaking
```

## Generated Code Location

Generated Go code will be in `pkg/proto/`:
- `pkg/proto/common/` - Common enums
- `pkg/proto/alerts/` - Alert messages
- `pkg/proto/rules/` - Rule messages
- `pkg/proto/notifications/` - Notification messages

## Using in Services

Each service needs to:

1. Add replace directive in `go.mod`:
   ```go
   replace github.com/afikmenashe/alerting-platform/pkg/proto => ../../pkg/proto
   ```

2. Add dependency:
   ```bash
   go get google.golang.org/protobuf/proto
   ```

3. Import in code:
   ```go
   import (
       "google.golang.org/protobuf/proto"
       pb "github.com/afikmenashe/alerting-platform/pkg/proto/alerts"
       pbcommon "github.com/afikmenashe/alerting-platform/pkg/proto/common"
   )
   ```

## Schema Evolution

When modifying `.proto` files:

1. **Never reuse field numbers** - Once used, a field number is reserved
2. **Use optional for new fields** - Maintains backward compatibility
3. **Deprecate before removing** - Mark fields as deprecated first
4. **Test compatibility** - Ensure old consumers can read new messages
5. **Check for breaking changes** - Run `make proto-breaking` before committing
6. **Verify generated code** - Run `make proto-verify-generated` to ensure code is current

See [Protobuf Tooling](../docs/architecture/PROTOBUF_TOOLING.md) for detailed tooling and validation documentation.

## Buf Configuration

This project uses `buf` for enhanced proto linting and breaking change detection:

- **`buf.yaml`** - Linting and breaking change rules
- **`buf.gen.yaml`** - Alternative code generation configuration (optional)

The configuration allows our flat directory structure (all .proto files in `proto/`) while enforcing best practices for naming, syntax, and compatibility.

## CI/CD Integration

Recommended checks for continuous integration:

```bash
# Validate proto syntax
make proto-validate

# Lint for best practices
make proto-lint

# Check for breaking changes
make proto-breaking

# Verify generated code is current
make proto-verify-generated
```

These checks ensure proto definitions stay valid and generated code stays synchronized.
