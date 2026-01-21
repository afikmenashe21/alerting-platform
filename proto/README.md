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

3. **buf** (optional, for linting)
   - macOS: `brew install bufbuild/buf/buf`
   - Or see: https://buf.build/docs/installation

## Generating Go Code

From the repository root:

```bash
# Install dependencies (first time only)
make proto-install-deps

# Generate Go code
make proto-generate

# Validate proto files
make proto-validate

# Lint proto files (optional, requires buf)
make proto-lint
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

See the [Protobuf Integration Strategy](../docs/architecture/PROTOBUF_INTEGRATION_STRATEGY.md) for migration details.
