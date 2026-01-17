# Rule-Updater Architecture

This document describes the architecture and design patterns used in the rule-updater service.

## Overview

The rule-updater service maintains a Redis snapshot of all enabled rules. It consumes `rule.changed` events from Kafka and rebuilds the snapshot whenever rules are created, updated, deleted, or disabled.

## Architecture Pattern

### Modular Design with Processor Pattern

The service uses a **Processor Pattern** to separate business logic from initialization:

```
cmd/rule-updater/main.go
├── Initialization (config, database, consumer, snapshot writer)
├── Initial snapshot build
└── Processor setup

internal/processor/
└── processor.go          # Rule change processing logic

internal/snapshot/
└── snapshot.go           # Snapshot building and Redis operations

internal/database/
└── database.go          # Data access layer

internal/consumer/
└── consumer.go         # Kafka consumer abstraction
```

## Directory Structure

```
rule-updater/
├── cmd/
│   └── rule-updater/
│       └── main.go              # CLI entry point, initialization
├── internal/
│   ├── processor/               # Processing orchestration
│   │   └── processor.go         # Rule change processing
│   ├── snapshot/                # Snapshot operations
│   │   └── snapshot.go         # Snapshot building and Redis ops
│   ├── database/               # Data access layer
│   │   └── database.go
│   ├── consumer/               # Kafka consumer
│   │   └── consumer.go
│   ├── events/                 # Event definitions
│   │   └── events.go
│   └── config/                 # Configuration
│       └── config.go
├── scripts/
│   └── run-all.sh
├── memory-bank/
├── Makefile
└── README.md
```

## Components

### Processor (`internal/processor/`)

The processor package orchestrates rule change processing:

- **ProcessRuleChanges**: Main processing loop
- **applyRuleChange**: Applies incremental updates to Redis

**Key Features:**
- Handles all rule change actions (CREATED, UPDATED, DELETED, DISABLED)
- Incremental updates using Lua scripts
- Error handling and offset management

### Snapshot (`internal/snapshot/`)

The snapshot package handles Redis snapshot operations:

- **BuildSnapshot**: Builds snapshot from rules
- **WriteSnapshot**: Writes snapshot to Redis with version increment
- **AddRuleDirect**: Adds/updates rule directly in Redis
- **RemoveRuleDirect**: Removes rule directly from Redis

**Key Features:**
- Atomic snapshot and version updates
- Direct Redis updates using Lua scripts
- Efficient memory usage

### Database (`internal/database/`)

The database package provides data access:

- **GetAllEnabledRules**: Fetches all enabled rules for initial snapshot
- **GetRule**: Fetches single rule for incremental updates

### Consumer (`internal/consumer/`)

The consumer package provides Kafka consumer abstraction:

- **ReadMessage**: Reads rule.changed events
- **CommitMessage**: Commits offsets after successful processing

## Design Patterns

### Processor Pattern

The processor pattern separates business logic from initialization:

```go
// Main initializes and delegates to processor
proc := processor.NewProcessor(consumer, db, writer)
proc.ProcessRuleChanges(ctx)
```

**Benefits:**
- Clear separation of concerns
- Testable business logic
- Easy to add new processing logic

### Incremental Update Pattern

The service uses incremental updates to avoid full snapshot rebuilds:

1. **CREATED/UPDATED**: Fetch rule from DB, add/update in Redis
2. **DELETED/DISABLED**: Remove rule from Redis
3. **Version Increment**: Atomic version bump with each update

**Benefits:**
- Fast updates (no full snapshot rebuild)
- Lower memory usage
- Atomic operations

## Processing Flow

1. **Startup**: Build initial snapshot from all enabled rules
2. **Event Consumption**: Read `rule.changed` events from Kafka
3. **Incremental Update**: Apply change directly to Redis
4. **Version Bump**: Increment `rules:version` atomically
5. **Offset Commit**: Commit Kafka offset after successful update

## Snapshot Format

The Redis snapshot contains:

- **Dictionaries**: Maps for severity, source, name (string → int)
- **Inverted Indexes**: Maps for bySeverity, bySource, byName (value → ruleInts)
- **Rule Metadata**: Maps ruleInt to rule_id, client_id
- **Version**: Monotonic integer version

## Error Handling

- **Database Errors**: Log and skip, Kafka will redeliver
- **Redis Errors**: Log and skip, Kafka will redeliver
- **Offset Commit Errors**: Log but continue processing

## Extensibility

### Adding New Processing Logic

1. Add method to `internal/processor/processor.go`
2. Call from `ProcessRuleChanges` if needed

### Adding New Snapshot Operations

1. Add method to `internal/snapshot/snapshot.go`
2. Use Lua scripts for atomic operations

## Testing

The modular architecture makes testing easier:

- **Processor Tests**: Mock consumer, database, and snapshot writer
- **Snapshot Tests**: Test snapshot building and Redis operations
- **Integration Tests**: Test full event processing flow
