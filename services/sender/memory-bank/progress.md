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

## AWS SES Migration (2026-01-23)
- [x] Migrated email sender from SMTP to AWS SES API
- [x] Using AWS SDK Go v2 (sesv2) for email delivery
- [x] IAM permissions added via task_role_policy_json in Terraform
- [x] Environment variables: AWS_REGION, SES_FROM (removed SMTP_*)
- [x] SES email identity verified in sandbox mode
- [x] Build requirement: `--platform linux/amd64` for ECS EC2

## Multi-Provider Email Architecture (2026-01-24)

### Strategy Pattern for Email Providers
Implemented a flexible email provider system using the Strategy Pattern to support multiple email backends with automatic fallback.

#### Architecture
```
internal/sender/email/
├── email.go              # Main sender (uses provider registry)
├── email_test.go         # Tests
└── provider/
    ├── provider.go       # Provider interface & registry
    ├── ses.go            # AWS SES implementation
    └── resend.go         # Resend implementation (default)
```

#### Supported Providers
| Provider | Status | Notes |
|----------|--------|-------|
| **Resend** | Primary (default) | Fast delivery, 3000 emails/month free tier, no verification needed |
| **AWS SES** | Fallback | Good for high volume, requires production access approval |

#### Configuration (Environment Variables)
| Variable | Description | Default |
|----------|-------------|---------|
| `EMAIL_PROVIDER` | Force specific provider (`resend`, `ses`) | Auto-detect |
| `EMAIL_FROM` | Sender email address | `onboarding@resend.dev` |
| `RESEND_API_KEY` | Resend API key | (required for Resend) |
| `AWS_REGION` | AWS region for SES | `us-east-1` |

#### Provider Selection Logic
1. If `EMAIL_PROVIDER` is explicitly set → use that provider
2. Auto-detect mode:
   - If `RESEND_API_KEY` is configured → use Resend
   - Otherwise → use SES
3. Fallback: If primary provider fails, automatically try other configured providers

#### Key Features
- **Strategy Pattern**: Easy to add new providers (Mailgun, SendGrid, etc.)
- **Automatic Fallback**: If primary provider fails, tries fallback providers
- **Provider Registry**: Centralized management of all email providers
- **HTML Emails**: Beautiful styled HTML email templates with severity-based colors
- **Extensible**: Implement `Provider` interface to add new backends

#### Provider Interface
```go
type Provider interface {
    Name() string
    Send(ctx context.Context, req *EmailRequest) error
    IsConfigured() bool
}
```

#### Adding a New Provider
1. Create `provider/newprovider.go` implementing `Provider` interface
2. Register in `email.go`: `registry.Register(provider.NewMyProvider())`
3. Add to fallback chain if desired

### Terraform Updates
- Added `email_provider`, `email_from`, `resend_api_key` variables
- Updated sender module environment variables
- Kept SES IAM permissions as fallback

## Retry & DLQ Implementation (2026-01-23)
- [x] Implemented retry with exponential backoff (3 attempts, 1s-30s)
- [x] Added jitter to prevent thundering herd
- [x] Simple DLQ pattern: mark notification as FAILED after all retries exhausted
- [x] Skip FAILED notifications on reprocessing (idempotency)

## Future Enhancements
- Rate limiting per client/endpoint
- Metrics and observability (success/failure rates per endpoint type)
- Support for Slack Block Kit for richer message formatting
- Additional email providers (Mailgun, SendGrid, Postmark)
