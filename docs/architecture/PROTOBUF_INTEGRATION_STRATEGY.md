# Protobuf Integration Strategy

## Executive Summary

This document outlines a comprehensive strategy for integrating Protocol Buffers (protobuf) into the alerting platform to replace JSON-based event serialization in Kafka messages. Protobuf will provide better performance, type safety, schema evolution, and reduced message sizes.

## Current State Analysis

### Current Serialization Approach
- **Format**: JSON with `schema_version` field for evolution
- **Serialization**: Go's `encoding/json` package (`json.Marshal`/`json.Unmarshal`)
- **Location**: All Kafka producer/consumer packages across services
- **Event Types**: 4 main event types across 4 Kafka topics

### Current Event Structures

#### 1. AlertNew (`alerts.new` topic)
- **Producer**: `alert-producer`
- **Consumer**: `evaluator`
- **Structure**: `Alert` struct with `alert_id`, `schema_version`, `event_ts`, `severity`, `source`, `name`, `context` (map[string]string)

#### 2. AlertMatched (`alerts.matched` topic)
- **Producer**: `evaluator`
- **Consumer**: `aggregator`
- **Structure**: Extends `AlertNew` with `client_id` and `rule_ids` ([]string)

#### 3. RuleChanged (`rule.changed` topic)
- **Producer**: `rule-service`
- **Consumer**: `rule-updater`, `evaluator` (optional)
- **Structure**: `rule_id`, `client_id`, `action` (enum), `version`, `updated_at`, `schema_version`

#### 4. NotificationReady (`notifications.ready` topic)
- **Producer**: `aggregator`
- **Consumer**: `sender`
- **Structure**: `notification_id`, `client_id`, `alert_id`, `schema_version`

### Current Code Locations

**Producers** (JSON serialization):
- `services/alert-producer/internal/producer/producer.go` (line 96)
- `services/evaluator/internal/producer/producer.go` (line 119)
- `services/aggregator/internal/producer/producer.go` (line 67)
- `services/rule-service/internal/producer/producer.go` (line 82)

**Consumers** (JSON deserialization):
- `services/evaluator/internal/consumer/consumer.go` (line 90)
- `services/evaluator/internal/ruleconsumer/ruleconsumer.go` (line 89)
- `services/aggregator/internal/consumer/consumer.go` (line 89)
- `services/rule-updater/internal/consumer/consumer.go` (line 90)
- `services/sender/internal/consumer/consumer.go` (line 89)

## Where Protobuf Should Be Integrated

### 1. Shared Protobuf Definitions (New)
**Location**: `proto/` directory at repository root

**Structure**:
```
proto/
├── alerts.proto          # AlertNew and AlertMatched messages
├── rules.proto           # RuleChanged message
├── notifications.proto   # NotificationReady message
└── common.proto          # Shared types (enums, common fields)
```

**Rationale**: Centralized schema definitions enable:
- Single source of truth for event schemas
- Cross-service type safety
- Schema evolution support
- Language-agnostic contracts (future polyglot support)

### 2. Event Serialization/Deserialization
**Replace**: All `json.Marshal`/`json.Unmarshal` calls in producer/consumer packages

**Services Affected**:
1. **alert-producer**: `internal/producer/producer.go`
2. **evaluator**: `internal/producer/producer.go`, `internal/consumer/consumer.go`, `internal/ruleconsumer/ruleconsumer.go`
3. **aggregator**: `internal/producer/producer.go`, `internal/consumer/consumer.go`
4. **rule-service**: `internal/producer/producer.go`
5. **rule-updater**: `internal/consumer/consumer.go`
6. **sender**: `internal/consumer/consumer.go`

### 3. Event Type Definitions
**Replace**: Current Go struct definitions in `internal/events/events.go` packages

**Services Affected**:
- `services/alert-producer/internal/generator/generator.go` (Alert struct)
- `services/evaluator/internal/events/events.go` (AlertNew, AlertMatched, RuleChanged)
- `services/aggregator/internal/events/events.go` (AlertMatched, NotificationReady)
- `services/rule-service/internal/events/events.go` (RuleChanged)
- `services/rule-updater/internal/events/events.go` (RuleChanged)
- `services/sender/internal/events/events.go` (NotificationReady)

**Approach**: Keep Go structs as wrappers around generated protobuf types, or migrate to generated types directly.

## What Protobuf Will Replace

### 1. JSON Serialization
**Current**:
```go
payload, err := json.Marshal(alert)
```

**Replaced with**:
```go
payload, err := proto.Marshal(alertProto)
```

### 2. JSON Deserialization
**Current**:
```go
var alert AlertNew
err := json.Unmarshal(msg.Value, &alert)
```

**Replaced with**:
```go
var alertProto pb.AlertNew
err := proto.Unmarshal(msg.Value, &alertProto)
```

### 3. Schema Version Field
**Current**: Manual `schema_version` integer field in JSON
**Replaced with**: Protobuf's built-in field numbering and backward compatibility

### 4. Type Definitions
**Current**: Go structs with JSON tags
**Replaced with**: Generated Go code from `.proto` files

## Benefits of Protobuf Integration

### 1. Performance Improvements
- **Smaller Message Size**: Protobuf is binary-encoded, typically 20-50% smaller than JSON
- **Faster Serialization**: Binary encoding is faster than JSON parsing
- **Reduced CPU Usage**: Less CPU overhead in high-throughput scenarios (evaluator, aggregator)

**Impact**: 
- Lower Kafka storage costs
- Reduced network bandwidth
- Better throughput in evaluator (matches thousands of alerts/second)
- Faster processing in aggregator (idempotent inserts)

### 2. Type Safety
- **Compile-time Validation**: Schema mismatches caught at compile time
- **Strong Typing**: Enums for `severity` and `action` fields
- **Field Validation**: Required vs optional fields enforced by protobuf

**Impact**:
- Fewer runtime errors
- Better developer experience
- IDE autocomplete support

### 3. Schema Evolution
- **Backward Compatibility**: Old consumers can read new messages (with new optional fields)
- **Forward Compatibility**: New consumers can read old messages
- **Field Deprecation**: Safe field removal with deprecation markers
- **Version Management**: Better than manual `schema_version` field

**Impact**:
- Zero-downtime deployments
- Gradual service upgrades
- No breaking changes during schema updates

### 4. Cross-Language Support
- **Polyglot Services**: Future services in Python, Java, etc. can use same schemas
- **Tooling**: Rich ecosystem of protobuf tools (protoc, buf, etc.)
- **Documentation**: Auto-generated API documentation from `.proto` files

**Impact**:
- Future flexibility for multi-language services
- Better tooling and validation

### 5. Better Developer Experience
- **Code Generation**: Auto-generated Go structs from `.proto` files
- **IDE Support**: Better autocomplete and type checking
- **Validation Tools**: `buf` for linting and breaking change detection

**Impact**:
- Faster development
- Fewer bugs
- Better maintainability

## Implementation Strategy

### Phase 1: Setup and Infrastructure (Foundation)

#### 1.1 Create Protobuf Definitions
**Location**: `proto/` directory

**Files to Create**:
- `proto/common.proto`: Shared enums (Severity, Action)
- `proto/alerts.proto`: AlertNew, AlertMatched messages
- `proto/rules.proto`: RuleChanged message
- `proto/notifications.proto`: NotificationReady message

**Key Design Decisions**:
- Use `string` for UUIDs (alert_id, rule_id, etc.) - standard practice
- Use `int64` for timestamps (Unix epoch)
- Use `map<string, string>` for context field
- Use `repeated string` for arrays (rule_ids)
- Use protobuf enums for `severity` and `action`

#### 1.2 Setup Build Tooling
**Add to root `Makefile`**:
- `make proto-generate`: Generate Go code from `.proto` files
- `make proto-validate`: Validate `.proto` files with `buf`
- `make proto-lint`: Lint `.proto` files

**Dependencies**:
- `google.golang.org/protobuf` (Go protobuf library)
- `buf` CLI tool (optional, for validation)
- `protoc` compiler with Go plugin

#### 1.3 Create Shared Proto Package
**Location**: `pkg/proto/` at repository root

**Purpose**: Generated Go code from `.proto` files, imported by all services

**Structure**:
```
pkg/
└── proto/
    ├── alerts/alerts.pb.go        # Generated from proto/alerts.proto
    ├── rules/rules.pb.go          # Generated from proto/rules.proto
    ├── notifications/notifications.pb.go # Generated from proto/notifications.proto
    └── common/common.pb.go        # Generated from proto/common.proto
```

### Phase 2: Service-by-Service Migration

#### 2.1 Migration Order (Low Risk → High Risk)

1. **rule-service** (Lowest risk - single producer, no critical path)
   - Migrate `RuleChanged` event
   - Update producer to use protobuf
   - Test with rule-updater (can run both JSON and protobuf consumers in parallel)

2. **alert-producer** (Low risk - single producer)
   - Migrate `AlertNew` event
   - Update producer to use protobuf
   - Test with evaluator (can support both formats during transition)

3. **evaluator** (Medium risk - critical path, but can support dual format)
   - Migrate `AlertNew` consumer (read protobuf)
   - Migrate `AlertMatched` producer (write protobuf)
   - Support both JSON and protobuf during transition period

4. **aggregator** (Medium risk - critical path)
   - Migrate `AlertMatched` consumer (read protobuf)
   - Migrate `NotificationReady` producer (write protobuf)
   - Support both formats during transition

5. **rule-updater** (Low risk - single consumer)
   - Migrate `RuleChanged` consumer (read protobuf)
   - No producer changes needed

6. **sender** (Low risk - single consumer)
   - Migrate `NotificationReady` consumer (read protobuf)
   - No producer changes needed

#### 2.2 Dual-Format Support (Transition Period)

**Strategy**: Support both JSON and protobuf formats during migration

**Implementation**:
- Detect message format via Kafka headers (`content-type: application/json` vs `application/x-protobuf`)
- Or detect via message prefix (JSON starts with `{`, protobuf is binary)
- Support both deserialization paths during transition

**Duration**: 1-2 weeks per service, allowing gradual rollout

### Phase 3: Code Changes Per Service

#### 3.1 Producer Changes Pattern

**Before**:
```go
payload, err := json.Marshal(alert)
msg := kafka.Message{
    Value: payload,
    Headers: []kafka.Header{
        {Key: "schema_version", Value: []byte(fmt.Sprintf("%d", alert.SchemaVersion))},
    },
}
```

**After**:
```go
alertProto := &pb.AlertNew{
    AlertId:       alert.AlertID,
    SchemaVersion: int32(alert.SchemaVersion),
    EventTs:       alert.EventTS,
    Severity:      pb.Severity(pb.Severity_value[alert.Severity]),
    Source:        alert.Source,
    Name:          alert.Name,
    Context:       alert.Context,
}
payload, err := proto.Marshal(alertProto)
msg := kafka.Message{
    Value: payload,
    Headers: []kafka.Header{
        {Key: "content-type", Value: []byte("application/x-protobuf")},
    },
}
```

#### 3.2 Consumer Changes Pattern

**Before**:
```go
var alert AlertNew
err := json.Unmarshal(msg.Value, &alert)
```

**After**:
```go
var alertProto pb.AlertNew
err := proto.Unmarshal(msg.Value, &alertProto)
// Convert to internal struct if needed
alert := &AlertNew{
    AlertID:       alertProto.AlertId,
    SchemaVersion: int(alertProto.SchemaVersion),
    EventTS:       alertProto.EventTs,
    Severity:      alertProto.Severity.String(),
    Source:        alertProto.Source,
    Name:          alertProto.Name,
    Context:       alertProto.Context,
}
```

### Phase 4: Testing Strategy

#### 4.1 Unit Tests
- Update all event serialization/deserialization tests
- Test backward compatibility (old JSON messages)
- Test forward compatibility (new protobuf fields)

#### 4.2 Integration Tests
- End-to-end tests with protobuf messages
- Dual-format support tests (JSON + protobuf)
- Schema evolution tests

#### 4.3 Performance Tests
- Benchmark protobuf vs JSON serialization
- Measure message size reduction
- Measure throughput improvement

### Phase 5: Cleanup and Optimization

#### 5.1 Remove JSON Support
- After all services migrated, remove JSON deserialization code
- Remove `schema_version` field (replaced by protobuf field numbers)
- Clean up unused JSON structs

#### 5.2 Optimize Generated Code
- Review generated protobuf code
- Add custom marshalers if needed for performance
- Optimize enum conversions

## Protobuf Schema Design

### proto/common.proto
```protobuf
syntax = "proto3";

package alerting.common;

option go_package = "github.com/afikmenashe/alerting-platform/pkg/proto/common";

// Severity levels for alerts
enum Severity {
  SEVERITY_UNSPECIFIED = 0;
  SEVERITY_LOW = 1;
  SEVERITY_MEDIUM = 2;
  SEVERITY_HIGH = 3;
  SEVERITY_CRITICAL = 4;
}

// Rule change actions
enum RuleAction {
  RULE_ACTION_UNSPECIFIED = 0;
  RULE_ACTION_CREATED = 1;
  RULE_ACTION_UPDATED = 2;
  RULE_ACTION_DELETED = 3;
  RULE_ACTION_DISABLED = 4;
}
```

### proto/alerts.proto
```protobuf
syntax = "proto3";

package alerting.alerts;

import "common.proto";

option go_package = "github.com/afikmenashe/alerting-platform/pkg/proto/alerts";

// AlertNew represents a new alert event (alerts.new topic)
message AlertNew {
  string alert_id = 1;                    // UUID v4
  int32 schema_version = 2;                // Schema version (currently 1)
  int64 event_ts = 3;                      // Unix timestamp (seconds)
  alerting.common.Severity severity = 4;   // Alert severity
  string source = 5;                      // Source system
  string name = 6;                         // Alert name/type
  map<string, string> context = 7;         // Optional context metadata
}

// AlertMatched represents a matched alert (alerts.matched topic)
message AlertMatched {
  string alert_id = 1;
  int32 schema_version = 2;
  int64 event_ts = 3;
  alerting.common.Severity severity = 4;
  string source = 5;
  string name = 6;
  map<string, string> context = 7;
  string client_id = 8;                   // Client this alert matched for
  repeated string rule_ids = 9;            // Rule IDs that matched
}
```

### proto/rules.proto
```protobuf
syntax = "proto3";

package alerting.rules;

import "common.proto";

option go_package = "github.com/afikmenashe/alerting-platform/pkg/proto/rules";

// RuleChanged represents a rule change event (rule.changed topic)
message RuleChanged {
  string rule_id = 1;                     // UUID of the rule
  string client_id = 2;                   // Client ID the rule belongs to
  alerting.common.RuleAction action = 3;  // CREATED, UPDATED, DELETED, DISABLED
  int32 version = 4;                      // Rule version (optimistic locking)
  int64 updated_at = 5;                   // Unix timestamp
  int32 schema_version = 6;               // Schema version (currently 1)
}
```

### proto/notifications.proto
```protobuf
syntax = "proto3";

package alerting.notifications;

option go_package = "github.com/afikmenashe/alerting-platform/pkg/proto/notifications";

// NotificationReady represents a notification ready event (notifications.ready topic)
message NotificationReady {
  string notification_id = 1;             // UUID of the notification
  string client_id = 2;                   // Client ID
  string alert_id = 3;                    // Alert ID
  int32 schema_version = 4;                // Schema version (currently 1)
}
```

## Migration Risks and Mitigation

### Risk 1: Breaking Changes During Migration
**Mitigation**:
- Implement dual-format support (JSON + protobuf)
- Gradual service-by-service migration
- Keep JSON support until all services migrated
- Feature flags for protobuf enablement

### Risk 2: Schema Evolution Issues
**Mitigation**:
- Follow protobuf best practices (never reuse field numbers, use optional for new fields)
- Use `buf` tool for breaking change detection
- Document schema evolution policy
- Version protobuf schemas in git

### Risk 3: Performance Regression
**Mitigation**:
- Benchmark before and after migration
- Test with production-like load
- Monitor Kafka message sizes and throughput
- Rollback plan if performance degrades

### Risk 4: Development Overhead
**Mitigation**:
- Automate code generation in Makefile
- Provide clear migration guide
- Update documentation
- Training for team on protobuf best practices

## Success Metrics

### Performance Metrics
- **Message Size Reduction**: Target 20-30% reduction vs JSON
- **Serialization Speed**: Target 2-3x faster than JSON
- **Throughput**: Maintain or improve current throughput

### Quality Metrics
- **Type Safety**: Zero runtime type errors (caught at compile time)
- **Schema Evolution**: Successful backward/forward compatibility tests
- **Error Rate**: No increase in deserialization errors

### Developer Experience
- **Build Time**: Minimal increase in build time (code generation)
- **Code Clarity**: Maintainable and readable generated code
- **Documentation**: Complete protobuf schema documentation

## Timeline Estimate

### Phase 1: Setup (1 week)
- Create `.proto` files
- Setup build tooling
- Generate Go code
- Initial testing

### Phase 2: Migration (4-6 weeks)
- Service-by-service migration
- Dual-format support
- Testing and validation
- Performance benchmarking

### Phase 3: Cleanup (1 week)
- Remove JSON support
- Update documentation
- Final optimization

**Total**: 6-8 weeks for complete migration

## Dependencies and Tools

### Required Tools
- `protoc` (Protocol Buffer Compiler)
- `protoc-gen-go` (Go plugin for protoc)
- `google.golang.org/protobuf` (Go protobuf library)
- `buf` (optional, for validation and linting)

### Go Dependencies
```go
require (
    google.golang.org/protobuf v1.32.0
)
```

## Next Steps

1. **Review and Approve Strategy**: Get team buy-in on approach
2. **Create Protobuf Definitions**: Start with `proto/common.proto` and `proto/alerts.proto`
3. **Setup Build Infrastructure**: Add Makefile targets and CI/CD integration
4. **Pilot Migration**: Start with `rule-service` (lowest risk)
5. **Iterate and Refine**: Adjust strategy based on pilot results
6. **Full Migration**: Roll out to remaining services

## References

- [Protocol Buffers Documentation](https://protobuf.dev/)
- [Go Protocol Buffers Guide](https://protobuf.dev/getting-started/gotutorial/)
- [Protobuf Best Practices](https://protobuf.dev/programming-guides/dos-donts/)
- [Schema Evolution Guide](https://protobuf.dev/programming-guides/proto3/#updating)
