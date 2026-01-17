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
├── slack/             # Slack sender module
├── webhook/           # Webhook sender module
└── payload/           # Payload builders
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

## Future Enhancements
- Retry logic with exponential backoff for failed sends
- Rate limiting per client/endpoint
- Dead letter queue for persistently failed sends
- Metrics and observability (success/failure rates per endpoint type)
- Support for additional email providers (SendGrid, SES) as alternatives to SMTP
- Support for Slack Block Kit for richer message formatting
