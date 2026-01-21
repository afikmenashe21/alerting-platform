# Protobuf Enum Design Decision

**Date**: 2026-01-21  
**Decision**: Align protobuf enum values with database format

## Background

After protobuf migration, alerts were not matching rules due to a severity format mismatch.

### Initial Problem

- **Database/Rules**: Store severity as `LOW`, `MEDIUM`, `HIGH`, `CRITICAL`
- **Protobuf enums** (initial): `SEVERITY_LOW`, `SEVERITY_MEDIUM`, `SEVERITY_HIGH`, `SEVERITY_CRITICAL`
- **Issue**: `.String()` returned `"SEVERITY_LOW"` instead of `"LOW"`, breaking rule matching

### Initial Fix Attempt

Added `severityToString()` helper to strip the `SEVERITY_` prefix:

```go
func severityToString(sev pbcommon.Severity) string {
    s := sev.String()
    return strings.TrimPrefix(s, "SEVERITY_")
}
```

**Problem**: Required conversion helpers in every consumer, two representations of same concept.

## Final Solution

Changed protobuf enum definitions to match database format directly.

### Proto Definition

```protobuf
// Severity levels for alerts
// Values match database enum (LOW, MEDIUM, HIGH, CRITICAL)
enum Severity {
  UNSPECIFIED = 0;
  LOW = 1;
  MEDIUM = 2;
  HIGH = 3;
  CRITICAL = 4;
}
```

### Generated Go Code

```go
const (
    Severity_UNSPECIFIED Severity = 0
    Severity_LOW         Severity = 1
    Severity_MEDIUM      Severity = 2
    Severity_HIGH        Severity = 3
    Severity_CRITICAL    Severity = 4
)

func (x Severity) String() string {
    return "LOW"  // or "MEDIUM", "HIGH", "CRITICAL"
}
```

### Usage in Code

**Producer** (alert-producer, evaluator):
```go
sev := pbcommon.Severity_UNSPECIFIED
switch strings.ToUpper(alert.Severity) {
case "LOW":
    sev = pbcommon.Severity_LOW
case "MEDIUM":
    sev = pbcommon.Severity_MEDIUM
// ...
}
```

**Consumer** (evaluator, aggregator):
```go
alert := &events.AlertNew{
    Severity: pb.Severity.String(),  // Returns "LOW", "MEDIUM", etc.
    // ...
}
```

## Benefits

1. **Single source of truth**: Database format is the canonical representation
2. **No conversion needed**: `.String()` returns the right format directly
3. **Simpler code**: Removed helper functions and extra imports
4. **Better readability**: Logs show `"LOW"` instead of `"SEVERITY_LOW"`
5. **Go scoping**: `pbcommon.Severity_LOW` is already clear and unambiguous

## Why This Works

- **Go enum scoping**: Enums in Go are scoped to their package, so `Severity_LOW` has no conflict risk
- **Database is source of truth**: CHECK constraint defines valid values: `('LOW', 'MEDIUM', 'HIGH', 'CRITICAL', '*')`
- **Protobuf wire format**: Uses integer values (1, 2, 3, 4) over the wire - enum names are just for readability
- **Style guide flexibility**: While protobuf style guide recommends prefixes (mainly for C++), Go's scoping makes this unnecessary

## Protobuf Style Guide Note

The official protobuf style guide recommends prefixing enum values to avoid naming conflicts:
- Example: `SEVERITY_LOW` instead of `LOW`
- Primary reason: C++ doesn't scope enums

**Our decision**: Prioritize simplicity and alignment with our database schema over style guide convention, since:
1. Go provides proper enum scoping
2. Database is our source of truth
3. Simpler code is more maintainable
4. This is an internal service, not a public API

## Files Changed

1. `proto/common.proto` - Updated Severity enum values
2. `pkg/proto/common/common.pb.go` - Regenerated (removed prefix)
3. `services/evaluator/internal/consumer/consumer.go` - Removed conversion helper
4. `services/aggregator/internal/consumer/consumer.go` - Removed conversion helper
5. `services/alert-producer/internal/producer/producer.go` - Updated enum references
6. `services/evaluator/internal/producer/producer.go` - Updated enum references

## Testing

- ✅ All services compile successfully
- ✅ All tests pass
- ✅ No linter errors
- ✅ Rule matching now works correctly

## Future Considerations

If adding new severity levels:
1. Add to database CHECK constraint
2. Add to protobuf enum
3. Add to producer switch statements
4. No other changes needed - the design is aligned
