# sender – Active Context

## Completed
- ✅ Kafka consumer for `notifications.ready` topic
- ✅ Database layer: read notification, update status, query all endpoint types by rule_ids
- ✅ Modular architecture with strategy pattern
- ✅ Email sender module using SMTP
- ✅ Slack sender module using Incoming Webhooks API
- ✅ Webhook sender module using HTTP POST
- ✅ Payload builder module for all channel types
- ✅ Strategy registry for extensible sender management
- ✅ Main processing loop with idempotency checks
- ✅ Graceful shutdown handling
- ✅ At-least-once delivery semantics

## Implementation Details

### Flow
1. Consume `notifications.ready` events from Kafka
2. Fetch notification from Postgres by `notification_id`
3. Check if already SENT (idempotency)
4. Query all endpoints (email, slack, webhook) for all rule_ids in the notification
5. Send notifications via appropriate channels (email via SMTP, Slack via webhook API, webhooks via HTTP POST)
6. Update notification status to SENT
7. Commit Kafka offset

### Database Queries
- `GetNotification`: Fetches notification with context (JSONB) and rule_ids (array)
- `GetEndpointsByRuleIDs`: Queries endpoints table for all enabled endpoints (email, slack, webhook) matching rule_ids
- `GetEmailEndpointsByRuleIDs`: Legacy method, now wraps `GetEndpointsByRuleIDs` and filters to email only
- `UpdateNotificationStatus`: Updates status to SENT (idempotent)

### Architecture

The sender service uses a **modular architecture with strategy pattern**:

```
internal/sender/
├── sender.go          # Coordinator that routes to strategies
├── strategy/          # Strategy interface and registry
│   └── strategy.go
├── email/              # Email sender implementation
│   └── email.go
├── slack/              # Slack sender implementation
│   └── slack.go
├── webhook/            # Webhook sender implementation
│   └── webhook.go
└── payload/            # Payload builders for all channels
    └── payload.go
```

### Sender Implementations

All senders implement the `NotificationSender` interface from the strategy package.

#### Email Sender (`internal/sender/email/`)
- Uses SMTP protocol for sending emails
- Configurable via environment variables: `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM`
- Defaults to localhost:1025 (common for local development with mailhog/mailcatcher)
- Builds RFC 822 formatted email messages
- Handles comma-separated email addresses

#### Slack Sender (`internal/sender/slack/`)
- Uses Slack Incoming Webhooks API
- Sends formatted messages with color-coded attachments based on severity
- Includes alert details, context, and metadata in structured format
- Supports rich formatting with fields and attachments

#### Webhook Sender (`internal/sender/webhook/`)
- Uses HTTP POST requests to webhook URLs
- Sends JSON payload with notification details
- Includes timestamp and all alert metadata
- Configurable timeout (30 seconds default)

#### Payload Builders (`internal/sender/payload/`)
- Centralized payload building for all channel types
- `BuildEmailPayload`: Email subject and body
- `BuildSlackPayload`: Slack webhook payload with attachments
- `BuildWebhookPayload`: Generic webhook JSON payload

### Strategy Pattern Implementation

- **Strategy Interface** (`strategy.NotificationSender`): Defines `Send()` and `Type()` methods
- **Strategy Registry** (`strategy.Registry`): Manages registration and retrieval of sender strategies
- **Coordinator** (`sender.Sender`): Routes notifications to appropriate strategies based on endpoint type
- **Extensibility**: New sender types can be added by implementing the interface and registering

### Multi-Channel Support
- `SendNotification` method routes to appropriate sender based on endpoint type using strategy pattern
- Groups endpoints by type (email, slack, webhook) to avoid duplicates
- Sends to all endpoint types sequentially (can be parallelized in future)
- Partial failures are logged but don't fail the entire operation (if at least one send succeeds)

### Idempotency
- Checks notification status before sending (skip if already SENT)
- If notification sent but status update fails, retry will skip (already sent check)
- At-least-once semantics: Kafka redelivery is safe

## Next Steps
- Add retry logic with exponential backoff for failed sends
- Add rate limiting per client/endpoint
- Add dead letter queue for persistently failed sends
- Add metrics and observability (success/failure rates per endpoint type)
- Support for additional email providers (SendGrid, SES) as alternatives to SMTP
