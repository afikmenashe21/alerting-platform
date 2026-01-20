# sender – Progress

## Completed
- [x] Kafka consumer for `notifications.ready` topic
- [x] Database layer: read notification, update status, query all endpoint types
- [x] Email sender implementation using SMTP
- [x] Slack sender implementation using Incoming Webhooks API
- [x] Webhook sender implementation using HTTP POST
- [x] Main processing loop with idempotency
- [x] Multi-channel notification routing
- [x] Makefile and docker-compose setup
- [x] Run script for easy setup

## Architecture Decisions

### Modular Architecture with Strategy Pattern
- **Strategy Pattern**: All senders implement `NotificationSender` interface
- **Separate Modules**: Each sender type (email, slack, webhook) in its own package
- **Payload Builders**: Centralized payload building in `payload` package
- **Strategy Registry**: Manages sender registration and retrieval
- **Coordinator Pattern**: Main `Sender` struct coordinates routing to strategies
- **Extensibility**: Easy to add new sender types by implementing interface and registering

### Directory Structure
```
internal/sender/
├── sender.go          # Coordinator
├── strategy/          # Interface and registry
├── email/             # Email sender module
│   ├── email.go       # Main sender struct, configuration, Send method
│   ├── smtp.go        # TLS connection handling
│   └── message.go     # Email message building
├── slack/             # Slack sender module
├── webhook/           # Webhook sender module
├── payload/           # Payload builders
└── validation/        # Shared validation utilities
```

### Multi-Channel Sender Design
- `Sender` coordinator routes to appropriate strategy based on endpoint type
- `SendNotification` method groups endpoints by type and routes to strategies
- Groups endpoints by type to avoid duplicate sends
- Partial failures are logged but don't fail entire operation if at least one send succeeds

### Email Sender
- Uses SMTP protocol (standard email delivery)
- Configurable via environment variables with sensible defaults
- Defaults to localhost:1025 for local development (compatible with mailhog/mailcatcher)
- Builds RFC 822 formatted email messages
- Collects unique email addresses from all matching rule endpoints

### Slack Sender
- Uses Slack Incoming Webhooks API
- Sends formatted messages with color-coded attachments
- Severity-based color coding: CRITICAL (red), HIGH/MEDIUM (yellow), LOW (green)
- Includes structured fields and context in attachments

### Webhook Sender
- Uses HTTP POST requests with JSON payload
- Configurable timeout (30 seconds default)
- Includes full notification details and timestamp
- Standard JSON format for easy integration

### Database Access
- Sender queries both `notifications` (data-plane) and `endpoints` (control-plane) tables
- Uses same Postgres database as other services
- `GetEndpointsByRuleIDs`: Efficient query using `ANY($1)` with array parameter for all endpoint types
- `GetEmailEndpointsByRuleIDs`: Legacy method maintained for backward compatibility

### Idempotency Strategy
- Check status before sending (skip if SENT)
- Update status after successful send
- If status update fails, retry will skip (already sent check)
- At-least-once delivery: safe to redeliver

### Error Handling
- Don't commit offset on error (Kafka will redeliver)
- Log errors with context (notification_id, rule_ids, endpoint types, etc.)
- Continue processing other messages on error
- Partial send failures are logged but don't fail the operation if at least one channel succeeds

## Code Cleanup and Modularization

### Shared Validation Helper
- **Extracted `isValidURL` function** from `slack.go` and `webhook.go` to shared `validation` package
- Created `internal/sender/validation/validation.go` for reusable validation utilities
- Removed code duplication between Slack and webhook senders

### Email Package Modularization
- **Split `email.go` (307 lines)** into three focused files:
  - `email.go` (164 lines): Main sender struct, configuration, and Send method
  - `smtp.go` (118 lines): TLS connection handling (sendWithTLS method)
  - `message.go` (40 lines): Email message building (buildEmailMessage, parseRecipients)
- All files now under 300 lines, organized by concern
- Maintained all existing functionality and test coverage

## Additional Code Cleanup and Modularization
- [x] Split large files by resource/concern:
  - `cmd/sender/main.go` (209 lines) → split into main.go (initialization) and processor.go (processing loop)
  - `internal/database/database.go` (220 lines) → split into database.go (connection management), notifications.go (notification operations), and endpoints.go (endpoint operations)
  - Improved separation of concerns: connection management, notification operations, and endpoint operations
  - All tests pass; behavior unchanged

## Future Enhancements
- Retry logic with exponential backoff for failed sends
- Rate limiting per client/endpoint
- Dead letter queue for persistently failed sends
- Metrics and observability (success/failure rates per endpoint type)
- Support for additional email providers (SendGrid, SES) as alternatives to SMTP
- Support for Slack Block Kit for richer message formatting
