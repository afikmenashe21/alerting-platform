# Evaluator Architecture

This document describes the architecture and design patterns used in the evaluator service. The evaluator is the **core matching engine** of the alerting platform, responsible for high-throughput rule evaluation.

## Overview

The evaluator service consumes alerts from Kafka, matches them against rules using in-memory indexes, and publishes matched alerts grouped by client. It uses hot-reload capabilities to update rule indexes when rules change without service restart.

### Why This Design Matters

The evaluator is designed for **high throughput** and **low latency**:
- **Stateless**: No database queries during alert processing (rules loaded from Redis snapshot)
- **In-memory indexes**: O(1) lookups for rule matching
- **Efficient intersection**: Minimizes computation by starting with smallest candidate set
- **Hot reload**: Rule updates without downtime or alert processing interruption
- **Tenant locality**: Output partitioned by `client_id` for downstream processing efficiency

## Architecture Pattern

### Modular Design with Processor Pattern

The service uses a **Processor Pattern** with separate components for alert processing and rule change handling:

```
cmd/evaluator/main.go
├── Initialization (config, Redis, snapshot loader)
├── Index building from snapshot
├── Rule change handler (background goroutine)
└── Alert processor setup

internal/processor/
├── processor.go          # Alert evaluation processing
└── rulehandler.go       # Rule change event handling

internal/matcher/
└── matcher.go           # Rule matching logic

internal/indexes/
└── indexes.go          # In-memory index management

internal/reloader/
└── reloader.go         # Hot reload mechanism
```

## Directory Structure

```
evaluator/
├── cmd/
│   └── evaluator/
│       └── main.go              # CLI entry point, initialization
├── internal/
│   ├── processor/               # Processing orchestration
│   │   ├── processor.go       # Alert evaluation processing
│   │   └── rulehandler.go     # Rule change event handling
│   ├── matcher/                # Rule matching
│   │   └── matcher.go
│   ├── indexes/                # In-memory indexes
│   │   └── indexes.go
│   ├── snapshot/               # Snapshot loading
│   │   └── snapshot.go
│   ├── reloader/               # Hot reload
│   │   └── reloader.go
│   ├── consumer/               # Kafka consumer
│   │   └── consumer.go
│   ├── ruleconsumer/          # Rule change consumer
│   │   └── ruleconsumer.go
│   ├── producer/              # Kafka producer
│   │   └── producer.go
│   └── config/                 # Configuration
│       └── config.go
├── scripts/
│   ├── run-all.sh
│   └── create-test-snapshot.go
├── memory-bank/
├── Makefile
└── README.md
```

## Components

### Processor (`internal/processor/`)

The processor package orchestrates alert evaluation:

- **ProcessAlerts**: Main alert processing loop
- **HandleRuleChanged**: Background handler for rule changes

**Key Features:**
- Dual processing: alerts and rule changes
- Clean separation of concerns
- Error handling and logging

### Matcher (`internal/matcher/`)

The matcher package provides rule matching logic:

- **Match**: Matches alert against rules using indexes
- **Intersection Algorithm**: Efficient set intersection

**Key Features:**
- Fast matching using inverted indexes
- Groups matches by client_id
- Handles wildcard rules

### Indexes (`internal/indexes/`)

The indexes package manages in-memory rule indexes. This is the **performance-critical component** that enables fast matching.

**Index Structure:**
- **Inverted Indexes**: Three maps for O(1) lookups:
  - `bySeverity[severity] → []ruleInt` - All rules matching a severity
  - `bySource[source] → []ruleInt` - All rules matching a source
  - `byName[name] → []ruleInt` - All rules matching a name
- **Rule Metadata**: `map[ruleInt]RuleInfo` - Maps integer ID to `{rule_id, client_id}`
- **Wildcard Handling**: Wildcard rules (`"*"`) appear in all candidate lists

**Why Integer-Based Indexing?**
- Memory efficient: Store strings once in dictionaries, use integers in indexes
- Fast comparisons: Integer equality checks vs string comparisons
- Compact serialization: Smaller Redis snapshots

**Thread Safety:**
- Indexes are **read-only** during alert processing
- Updates happen via **atomic pointer swap** (no locks needed)
- Hot reload builds new indexes, then atomically swaps the pointer

**Key Features:**
- Fast O(1) lookups using Go maps
- Memory-efficient integer-based indexing
- Thread-safe operations via atomic swaps
- Supports wildcard rules seamlessly

### Reloader (`internal/reloader/`)

The reloader package handles hot reloading:

- **Version Polling**: Polls Redis for version changes
- **ReloadNow**: Immediate reload from Redis snapshot
- **Atomic Swap**: Swaps indexes atomically

**Key Features:**
- Hot reload without service restart
- Version-based change detection
- Atomic index updates

## Design Patterns

### Processor Pattern

The processor pattern separates business logic from initialization:

```go
// Main initializes and delegates to processors
alertProc := processor.NewProcessor(consumer, producer, matcher)
ruleHandler := processor.NewRuleHandler(ruleConsumer, reloader)

go ruleHandler.HandleRuleChanged(ctx)
alertProc.ProcessAlerts(ctx)
```

**Benefits:**
- Clear separation of concerns
- Testable business logic
- Easy to add new processing logic

### Dual Processing Pattern

The service processes two types of events:

1. **Alerts**: High-throughput alert evaluation
2. **Rule Changes**: Low-frequency rule index updates

**Benefits:**
- Independent processing paths
- Different error handling strategies
- Optimal resource usage

## Matching Algorithm

The matching algorithm is the heart of the evaluator. It uses **inverted indexes** and **set intersection** to efficiently find all rules that match an alert.

### Step-by-Step Process

1. **Get Candidates**: For each alert field (severity, source, name), lookup candidate rules from inverted indexes:
   - `bySeverity[alert.severity]` → list of ruleInts matching this severity
   - `bySource[alert.source]` → list of ruleInts matching this source
   - `byName[alert.name]` → list of ruleInts matching this name
   - Wildcard rules (`"*"`) are included in all candidate lists

2. **Intersect Efficiently**: 
   - Start with the **smallest candidate list** (minimizes work)
   - For each ruleInt in the smallest list, check if it exists in the other two lists
   - Use boolean sets for O(1) membership checks
   - Result: list of ruleInts that match all three fields

3. **Group by Client**: 
   - Map matched ruleInts to their `client_id` and `rule_id`
   - Group by `client_id` (one client can have multiple matching rules)
   - Result: `map[client_id][]rule_id`

4. **Publish**: 
   - Emit **one message per client_id** to `alerts.matched` topic
   - Each message contains the full alert + `client_id` + all matching `rule_ids[]`
   - Messages are keyed by `client_id` for tenant locality (partitioning)

### Example

Given an alert: `{severity: "HIGH", source: "api", name: "timeout"}`

1. Candidates:
   - `bySeverity["HIGH"]` → `[1, 3]` (ruleInts 1 and 3)
   - `bySource["api"]` → `[1, 3]` (ruleInts 1 and 3)
   - `byName["timeout"]` → `[1, 3]` (ruleInts 1 and 3)

2. Intersection: Start with smallest list (all same size here), check membership → `[1, 3]`

3. Group by client:
   - RuleInt 1 → `{client_id: "client-1", rule_id: "rule-001"}`
   - RuleInt 3 → `{client_id: "client-2", rule_id: "rule-003"}`
   - Result: `{"client-1": ["rule-001"], "client-2": ["rule-003"]}`

4. Publish: Two messages (one for each client)

## Hot Reload Flow

The evaluator supports **zero-downtime rule updates** through a combination of version polling and event-driven reloads.

### Startup Sequence

1. **Load Snapshot**: Read `rules:snapshot` from Redis (JSON serialized indexes)
2. **Build Indexes**: Deserialize and construct in-memory inverted indexes
3. **Start Version Poller**: Background goroutine polls `rules:version` every N seconds
4. **Start Rule Consumer**: Background goroutine consumes `rule.changed` events
5. **Start Alert Processor**: Main loop begins processing alerts

### Runtime Reload (Two Mechanisms)

**Mechanism 1: Version Polling (Backup)**
- Polls Redis `rules:version` every N seconds (default: 5s)
- If version changed since last check → trigger reload
- Ensures eventual consistency even if events are missed

**Mechanism 2: Event-Driven (Primary)**
- Consumes `rule.changed` events from Kafka
- Immediately triggers reload on rule change
- Lower latency than polling

### Reload Process

1. **Load New Snapshot**: Read latest `rules:snapshot` from Redis
2. **Build New Indexes**: Construct new in-memory indexes (parallel to old ones)
3. **Atomic Swap**: Replace pointer to indexes atomically
4. **Garbage Collection**: Old indexes are garbage collected when no longer referenced

**Key Properties:**
- **Zero Downtime**: Alert processing continues during reload
- **Atomic**: Index swap is atomic (no partial updates)
- **No Locks**: Read-only access means no locking needed
- **Memory Efficient**: Old indexes GC'd automatically

## Error Handling

The evaluator uses different error handling strategies for different failure modes:

### Alert Processing Errors
- **Strategy**: Log and continue, rely on Kafka redelivery
- **Rationale**: Transient errors (network, parsing) should not stop processing
- **At-least-once**: Kafka will redeliver failed messages
- **Downstream**: Aggregator handles duplicates via idempotency

### Rule Change Errors
- **Strategy**: Log and continue, polling will catch up
- **Rationale**: Rule updates are eventually consistent
- **Fallback**: Version polling ensures eventual consistency
- **Impact**: Temporary rule staleness (acceptable for MVP)

### Index Build Errors
- **Strategy**: Log and exit (critical error)
- **Rationale**: Cannot process alerts without valid indexes
- **Recovery**: Service restart will reload from Redis
- **Monitoring**: Should alert on index build failures

### Snapshot Loading Errors
- **Startup**: Exit if initial snapshot fails (cannot start without rules)
- **Runtime**: Keep old indexes, log error, retry on next poll/event

## Extensibility

### Adding New Matching Logic

1. Add method to `internal/matcher/matcher.go`
2. Update `internal/processor/processor.go` to use it

### Adding New Index Types

1. Add index structure to `internal/indexes/indexes.go`
2. Update snapshot loading and building

## Data Flow

### Input: `alerts.new` Topic
- **Partitioning**: Keyed by `alert_id` (even distribution)
- **Consumer Group**: `evaluator-group` (one partition per instance)
- **Offset Management**: Commits after successful processing

### Output: `alerts.matched` Topic
- **Partitioning**: Keyed by `client_id` (tenant locality)
- **Format**: One message per client_id per alert
- **Benefits**: 
  - Aggregator can process all alerts for a client from same partition
  - Enables future client-based sharding
  - Better cache locality

### Rule Updates: `rule.changed` Topic
- **Partitioning**: Keyed by `rule_id`
- **Consumer Group**: `evaluator-rule-changed-group`
- **Purpose**: Immediate notification of rule changes

## Performance Characteristics

### Throughput
- **Target**: Process thousands of alerts per second
- **Bottlenecks**: 
  - Kafka consumer throughput
  - Index intersection computation
  - Kafka producer throughput
- **Optimizations**:
  - In-memory indexes (no I/O during matching)
  - Efficient intersection algorithm
  - Batch Kafka operations

### Latency
- **Target**: Sub-100ms p99 latency
- **Components**:
  - Kafka consumer poll: ~10-50ms
  - Index lookup + intersection: <1ms
  - Kafka producer: ~10-50ms
- **Total**: ~20-100ms per alert

### Memory
- **Indexes**: O(R) where R = number of rules
- **Rule Metadata**: O(R) for rule mappings
- **Dictionaries**: O(unique values) for string compression
- **Typical**: ~1-10MB for thousands of rules

## Testing

The modular architecture makes testing easier:

- **Processor Tests**: Mock consumer, producer, and matcher
- **Matcher Tests**: Test matching logic with sample data
- **Index Tests**: Test index building and lookups
- **Integration Tests**: Test full alert processing flow
- **Load Tests**: Measure throughput and latency under load
