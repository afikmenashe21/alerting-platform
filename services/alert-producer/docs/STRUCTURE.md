# Alert-Producer Architecture

This document explains the code organization, directory structure, and design decisions.

## 📁 Directory Structure

```
alert-producer/
├── .gitignore              # Git ignore rules
├── .dockerignore           # Docker ignore rules
├── go.mod                  # Go module definition
├── go.sum                  # Dependency lock file
├── Makefile                # Build automation
├── README.md               # Main documentation
│
├── cmd/                    # Application entry points
│   └── alert-producer/
│       └── main.go         # CLI entry point
│
├── internal/               # Private application code
│   ├── config/             # Configuration management
│   │   ├── config.go       # Config struct & validation
│   │   └── config_test.go  # Config unit tests
│   ├── generator/          # Alert generation
│   │   ├── generator.go    # Alert generator logic
│   │   └── generator_test.go # Generator unit tests
│   ├── processor/          # Processing orchestration
│   │   └── processor.go    # Main processor with mode handlers
│   └── producer/           # Kafka publishing
│       ├── producer.go     # Kafka producer wrapper
│       └── mock_producer.go # Mock for testing
│
├── docs/                   # 📚 All documentation
│   ├── README.md           # Documentation index
│   ├── SETUP_AND_RUN.md    # Getting started guide
│   ├── EVENT_STRUCTURE.md  # Alert event schema
│   ├── PARTITIONING.md     # Partitioning strategy
│   └── STRUCTURE.md        # This file
│
├── scripts/                # Utility scripts
│   ├── run-all.sh          # Run script (verifies centralized infrastructure)
│   └── test-producer.sh    # Integration test script
│
├── memory-bank/            # Project context (Memory Bank pattern)
│   ├── projectbrief.md     # Service brief
│   ├── techContext.md      # Technology choices
│   ├── systemPatterns.md   # Service patterns
│   ├── activeContext.md    # Current work status
│   └── progress.md         # Completed work
│
└── bin/                    # Build artifacts (gitignored)
    └── alert-producer      # Compiled binary
```

## 📁 Root Directory

### `go.mod` & `go.sum`
- **Purpose**: Go module definition and dependency lock file
- **Why**: Defines the module name (`alert-producer`) and all dependencies
- **Dependencies**:
  - `github.com/google/uuid` - Generate unique alert IDs
  - `github.com/segmentio/kafka-go` - Kafka producer client

### `Makefile`
- **Purpose**: Build automation and common tasks
- **Why**: Standardizes build, test, and run commands
- **Key targets**:
  - `make build` - Compiles the binary
  - `make run` - Builds and runs the service
  - `make test` - Runs unit tests
  - `make kafka-up` - Starts local Kafka
  - `make kafka-down` - Stops Kafka

### Root Files
- **`go.mod` & `go.sum`**: Go module definition and dependency lock
- **`Makefile`**: Build automation and common tasks
- **`README.md`**: Main documentation entry point
- **`.gitignore` & `.dockerignore`**: Ignore rules

**Note**: Infrastructure (Kafka, Postgres, Redis) is managed centrally at the project root level, not in this service directory.

---

## 📁 `cmd/` - Application Entry Points

### `cmd/alert-producer/main.go`
- **Purpose**: CLI entry point - the `main()` function
- **Why**: 
  - Parses command-line flags
  - Initializes components (config, generator, producer)
  - Orchestrates the main loop (burst or continuous mode)
  - Handles graceful shutdown (SIGINT/SIGTERM)
- **Responsibilities**:
  - Flag parsing and validation
  - Service lifecycle management
  - Error handling and logging
  - Mode selection (burst vs continuous)

**Why separate from `internal/`?**
- `cmd/` contains only entry points (main functions)
- Follows Go standard project layout
- Allows multiple binaries from same codebase (if needed)

---

## 📁 `internal/` - Private Application Code

The `internal/` directory contains code that is **not meant to be imported by other projects**. This enforces encapsulation.

### `internal/config/` - Configuration Management

#### `config.go`
- **Purpose**: Configuration struct and validation
- **Why**: 
  - Centralizes all configuration parameters
  - Validates config before service starts
  - Parses distribution strings (severity/source/name)
- **Key functions**:
  - `Config.Validate()` - Ensures all required fields are valid
  - `ParseDistribution()` - Parses "KEY:PERCENT,KEY:PERCENT" format

#### `config_test.go`
- **Purpose**: Unit tests for config package
- **Why**: Ensures validation logic works correctly
- **Tests**: Distribution parsing, config validation edge cases

**Why separate package?**
- Single responsibility: configuration only
- Reusable validation logic
- Easy to test in isolation

---

### `internal/generator/` - Alert Generation

#### `generator.go`
- **Purpose**: Creates synthetic alert events
- **Why**: 
  - Generates alerts with configurable distributions
  - Supports deterministic mode (via seed)
  - Creates realistic test data
- **Key components**:
  - `Alert` struct - The event schema
  - `Generator` - Main generation logic
  - `weightedValue` - Distribution weights
- **Key functions**:
  - `New()` - Creates generator with distributions
  - `Generate()` - Creates a new alert
  - `selectWeighted()` - Picks value from weighted distribution

#### `generator_test.go`
- **Purpose**: Unit tests for alert generation
- **Why**: Ensures alerts are generated correctly with proper distributions

**Why separate package?**
- Alert generation is independent of Kafka/producer
- Can be tested without Kafka
- Could be reused in other contexts (e.g., test fixtures)

---

### `internal/producer/` - Kafka Publishing

#### `producer.go`
- **Purpose**: Kafka producer wrapper
- **Why**: 
  - Abstracts Kafka client details
  - Handles message serialization
  - Manages partition key hashing
  - Auto-creates topics if needed
- **Key components**:
  - `AlertPublisher` interface - Abstraction for publishing
  - `Producer` - Real Kafka implementation
- **Key functions**:
  - `New()` - Creates Kafka producer, auto-creates topic
  - `Publish()` - Serializes and publishes alert
  - `hashAlertID()` - Creates partition key from alert_id
  - `createTopicIfNotExists()` - Topic management

#### `mock_producer.go`
- **Purpose**: Mock implementation for testing without Kafka
- **Why**: 
  - Allows testing without Kafka running
  - Useful for development/debugging
  - Logs alerts instead of publishing
- **Usage**: `--mock` flag enables this

**Why separate package?**
- Kafka integration is isolated
- Can swap implementations (real vs mock)
- Interface allows dependency injection

---

## 📁 `bin/` - Build Artifacts

### `bin/alert-producer`
- **Purpose**: Compiled binary (created by `make build`)
- **Why**: Executable that can be run directly
- **Note**: This is generated, not committed to git (should be in `.gitignore`)

---

## 📁 `scripts/` - Utility Scripts

### `scripts/run-all.sh`
- **Purpose**: Run script using centralized infrastructure
- **Why**: Infrastructure is managed centrally (via root-level docker-compose)
- **What it does**: 
  - Verifies infrastructure dependencies are running
  - Downloads Go dependencies
  - Builds the service
  - Runs the service
- **Note**: Infrastructure should be started from root with `make setup-infra`

### `scripts/test-producer.sh`
- **Purpose**: Integration test script
- **Why**: 
  - Automates end-to-end testing
  - Verifies Kafka connectivity
  - Validates message structure
  - Tests both burst and continuous modes
- **What it does**:
  1. Checks if Kafka is running
  2. Creates topic if needed
  3. Runs producer in burst mode
  4. Verifies messages in Kafka
  5. Validates JSON structure

---

## 📁 `memory-bank/` - Project Documentation

This follows the Memory Bank pattern for maintaining project context.

### `projectbrief.md`
- **Purpose**: Service-specific project brief
- **Why**: Documents what this service does and its features

### `techContext.md`
- **Purpose**: Technology choices for this service
- **Why**: Documents Go version, Kafka client, config approach

### `systemPatterns.md`
- **Purpose**: Service-specific patterns
- **Why**: Documents partitioning strategy, alert ID generation

### `activeContext.md`
- **Purpose**: Current work status
- **Why**: Tracks what we're working on right now

### `progress.md`
- **Purpose**: Completed work and decisions
- **Why**: Historical record of what's been built

**Why Memory Bank?**
- Maintains context across sessions
- Documents decisions for future reference
- Helps AI assistants understand the project

---

## 🏗️ Architecture Overview

```
┌─────────────────────────────────────────┐
│         cmd/alert-producer/main.go       │  ← Entry point
│  (Flag parsing, orchestration, logging) │
└──────────────┬──────────────────────────┘
               │
       ┌───────┴────────┐
       │                │
┌──────▼──────┐  ┌──────▼──────┐  ┌──────────────┐
│   config/   │  │  generator/ │  │  producer/  │
│             │  │             │  │             │
│ - Validate  │  │ - Generate  │  │ - Publish   │
│ - Parse     │  │ - Weighted  │  │ - Hash key  │
│   dists     │  │   selection │  │ - Kafka     │
└─────────────┘  └─────────────┘  └──────────────┘
```

## 📋 Design Principles

### 1. **Separation of Concerns**
- Each package has a single responsibility
- `config/` = configuration only
- `generator/` = alert generation only
- `producer/` = Kafka publishing only

### 2. **Testability**
- Each package has unit tests
- Mock producer for testing without Kafka
- Isolated components are easy to test

### 3. **Modularity**
- Packages can be used independently
- Interface-based design (`AlertPublisher`)
- Easy to swap implementations

### 4. **Go Best Practices**
- Standard project layout (`cmd/`, `internal/`)
- `internal/` prevents external imports
- Clear package boundaries

### 5. **Documentation**
- Inline comments explain complex logic
- Package-level documentation
- Separate docs for different concerns

## 🔄 Data Flow

```
1. main.go parses flags → config.Config
2. config.Validate() → ensures valid config
3. generator.New(config) → creates Generator
4. producer.New(brokers, topic) → creates Producer
5. Loop:
   - generator.Generate() → creates Alert
   - producer.Publish(alert) → hashes alert_id, serializes, sends to Kafka
```

## 🎯 Why This Structure?

1. **Scalability**: Easy to add new features (new packages)
2. **Maintainability**: Clear boundaries, easy to find code
3. **Testability**: Each component can be tested independently
4. **Reusability**: Components can be reused (e.g., generator for tests)
5. **Standards**: Follows Go community conventions

This structure makes the codebase:
- ✅ Easy to understand
- ✅ Easy to test
- ✅ Easy to extend
- ✅ Easy to maintain
