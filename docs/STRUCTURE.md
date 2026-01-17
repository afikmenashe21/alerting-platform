# Documentation Structure

This document describes the organization of the project documentation and root directory.

## Root Directory Structure

```
alerting-platform/
├── README.md                 # Main project overview and quick start
├── Makefile                  # Root-level build and run commands
├── docker-compose.yml        # Centralized infrastructure definition
├── .gitignore               # Git ignore rules
│
├── docs/                     # 📚 All documentation
│   ├── README.md            # Documentation index
│   ├── guides/              # Step-by-step guides
│   │   ├── SETUP.md        # Complete setup guide
│   │   └── QUICKSTART.md   # Quick start guide
│   ├── architecture/        # Architecture documentation
│   │   ├── INFRASTRUCTURE.md
│   │   └── MIGRATION_TO_CENTRALIZED.md
│   └── features/            # Feature-specific docs
│       ├── WILDCARD_RULES_DESIGN.md
│       └── WILDCARD_RULES_USAGE.md
│
├── services/                 # All Go services
│   ├── alert-producer/
│   ├── rule-service/
│   ├── rule-updater/
│   ├── evaluator/
│   ├── aggregator/
│   └── sender/
│
├── scripts/                  # Centralized utility scripts
│   ├── setup-infrastructure.sh
│   ├── verify-dependencies.sh
│   ├── run-migrations.sh
│   ├── create-kafka-topics.sh
│   ├── run-all-services.sh
│   └── test-data/           # Test data generation
│
├── memory-bank/              # Project memory bank (design decisions)
│   ├── projectbrief.md
│   ├── techContext.md
│   ├── systemPatterns.md
│   ├── activeContext.md
│   └── progress.md
│
├── migrations/               # Migration strategy documentation
│   └── MIGRATION_STRATEGY.md
│
└── rule-service-ui/         # React UI application
    ├── src/
    └── ...
```

## Documentation Categories

### Guides (`docs/guides/`)
Step-by-step guides for getting started:
- **SETUP.md** - Complete setup instructions
- **QUICKSTART.md** - Quick start for rapid setup

### Architecture (`docs/architecture/`)
Architecture and infrastructure documentation:
- **INFRASTRUCTURE.md** - Centralized infrastructure management
- **MIGRATION_TO_CENTRALIZED.md** - Migration guide

### Features (`docs/features/`)
Feature-specific documentation:
- **WILDCARD_RULES_DESIGN.md** - Wildcard rules design
- **WILDCARD_RULES_USAGE.md** - Wildcard rules usage

## Key Files

### Root Level
- **README.md** - Main entry point, overview, and quick start
- **Makefile** - Root-level commands (`make run-all`, `make setup-infra`, etc.)
- **docker-compose.yml** - Centralized infrastructure (Postgres, Kafka, Redis, etc.)

### Documentation
- **docs/README.md** - Documentation index and navigation
- **memory-bank/** - Design decisions, patterns, and project context
- **migrations/MIGRATION_STRATEGY.md** - Database migration strategy

## Finding Documentation

- **Getting Started?** → `README.md` → `docs/guides/QUICKSTART.md`
- **Setting Up?** → `docs/guides/SETUP.md`
- **Understanding Infrastructure?** → `docs/architecture/INFRASTRUCTURE.md`
- **Learning About Features?** → `docs/features/`
- **Understanding Design Decisions?** → `memory-bank/`
