# Alerting Platform – Progress

## Decisions made
- Kafka for replay + at-least-once.
- Postgres for control-plane + notification idempotency boundary.
- Redis snapshot for evaluator warmup.
- JSON contracts, add schema_version.

## Milestones
- [x] Topic contracts finalized (JSON structs + sample payloads)
- [x] Postgres migrations: clients/rules (rule-service)
- [x] rule-service: CRUD + publish rule.changed
- [x] rule-updater: rebuild snapshot → Redis + bump version
- [x] evaluator: warmup + match + publish alerts.matched (one message per client_id)
- [x] aggregator: dedupe insert + publish notifications.ready
- [x] sender: consume notifications.ready + send via email (SMTP), Slack (webhook API), and webhook (HTTP POST) + update status
- [x] alert-producer: generate alerts + load tests
- [x] rule-service-ui: React UI for CRUD operations on clients, rules, and endpoints

## Code health
- [x] rule-service: code cleanup and modularization:
  - Removed redundant code via private helpers (validation, HTTP, JSON parsing)
  - Modularized handlers: split 674-line `handlers.go` into resource-specific files:
    - `clients.go`, `rules.go`, `endpoints.go`, `notifications.go`
  - Added `publishRuleChangedEvent()` helper to reduce duplication
  - All rule-service tests pass; behavior unchanged.
- [x] rule-updater: code cleanup and modularization:
  - Split 618-line `snapshot.go` into three focused files:
    - `snapshot.go` (267 lines): Core Snapshot struct and in-memory operations
    - `writer.go` (156 lines): Writer struct and Redis operations
    - `lua_scripts.go` (207 lines): Lua script constants for direct Redis updates
  - Extracted redundant code into helper functions:
    - `getMaxDictValue()`: Reusable dictionary max value calculation
    - `removeFromIndex()`: Unified index removal logic
    - `newEmptySnapshot()`: Centralized empty snapshot creation
  - All tests pass; behavior unchanged.
- [x] sender: code cleanup and modularization:
  - Extracted duplicate `isValidURL` function from `slack.go` and `webhook.go` to shared `validation` package
  - Split 307-line `email.go` into three focused files:
    - `email.go` (164 lines): Main sender struct, configuration, and Send method
    - `smtp.go` (118 lines): TLS connection handling
    - `message.go` (40 lines): Email message building
  - All sender tests pass; behavior unchanged.

## Recent Decisions
- **Evaluator output format**: One message per client_id (not one message with all matches)
  - Enables tenant locality: messages partitioned by client_id
  - Simplifies aggregator: one client per message
  - If alert matches N clients, N messages are published

- **Sender service design**: Multi-channel sender supporting email (SMTP), Slack (Incoming Webhooks), and webhooks (HTTP POST)
  - Queries endpoints table for all endpoint types (email, slack, webhook) by rule_ids
  - Routes to appropriate sender based on endpoint type
  - Email: SMTP protocol with configurable server (defaults to localhost:1025 for local dev)
  - Slack: Incoming Webhooks API with formatted messages and severity-based color coding
  - Webhook: HTTP POST with JSON payload containing full notification details
  - Idempotent: checks status before sending, updates after
  - At-least-once delivery: safe to redeliver (skips if already SENT)
  - Partial failures are logged but don't fail operation if at least one channel succeeds

- **rule-service-ui**: React + Vite UI for rule-service management
  - Full CRUD operations for clients, rules, and endpoints
  - Modern UI with tabbed navigation
  - Connects to rule-service API at http://localhost:8081 (via Vite proxy)
  - Features: create, read, update, delete, toggle enable/disable for all entities
  - Optimistic locking support for rule updates (version field)

- **Centralized Infrastructure Management**: Created unified dependency management
  - Root `docker-compose.yml` for shared infrastructure (Postgres, Kafka, Redis, Zookeeper)
  - Centralized verification script (`scripts/infrastructure/verify-dependencies.sh`)
  - Centralized migration runner (`scripts/migrations/run-migrations.sh`)
  - Centralized Kafka topic creation (`scripts/infrastructure/create-kafka-topics.sh`)
  - Services should verify dependencies, NOT manage them
  - `make run-all` automatically starts infrastructure if not running (one-command startup)
  - See `docs/architecture/INFRASTRUCTURE.md` for full documentation

- **Security: Email Credentials**: Removed all hardcoded email credentials from documentation
  - Replaced hardcoded Gmail credentials in `GMAIL_SETUP.md`, `README.md`, and `TROUBLESHOOTING.md` with placeholders
  - Created `.env.example` template file for secure credential management
  - Code already uses environment variables correctly (no code changes needed)
  - All credentials must now be provided via environment variables or `.env` file
  - `.env` files are already in `.gitignore` to prevent accidental commits

- **UI Integration for alert-producer (Planned)**: Integration with rule-service-ui for alert generation
  - HTTP API wrapper around alert-producer functionality
  - New UI component in rule-service-ui for generating alerts
  - Support for all CLI configuration options via web interface
  - Real-time status monitoring and job tracking
  - Preset configurations and manual configuration options
  - See `services/alert-producer/memory-bank/` for detailed design
