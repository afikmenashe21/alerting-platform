# Sender

Delivers notifications via email (SMTP/Resend/SES), Slack, and webhooks. Final step in the alerting pipeline.

## Role in Pipeline

```
notifications.ready (Kafka) → [sender] → Email / Slack / Webhook
                                  ↕
                           Postgres (notifications + endpoints)
```

The sender reads notification details from the database, resolves delivery endpoints for the matching rules, sends via the appropriate channel, and updates the notification status to `SENT`.

## How It Works

1. Consumes `notifications.ready` messages from Kafka
2. Fetches the notification record from Postgres
3. Checks if already `SENT` (idempotency guard)
4. Queries `endpoints` table for all enabled endpoints matching the notification's `rule_ids`
5. Sends via the appropriate channel (email, Slack, webhook) using a strategy pattern
6. Updates notification status to `SENT`
7. Commits Kafka offset

## Delivery Channels

| Channel | Implementation | Configuration |
|---------|---------------|---------------|
| **Email** | SMTP, Resend API, AWS SES | `SMTP_*` or `RESEND_API_KEY` or `AWS_*` env vars |
| **Slack** | Webhook POST | Endpoint value = webhook URL |
| **Webhook** | HTTP POST with JSON payload | Endpoint value = target URL |

### Email Configuration

```bash
# SMTP (Gmail, MailHog, etc.)
export SMTP_HOST=smtp.gmail.com
export SMTP_PORT=587
export SMTP_USER=you@gmail.com
export SMTP_PASSWORD=app-password
export SMTP_FROM=you@gmail.com

# Or Resend API
export RESEND_API_KEY=re_...
export EMAIL_PROVIDER=resend

# Or AWS SES
export AWS_REGION=us-east-1
export EMAIL_PROVIDER=ses
```

For local testing, MailHog is included in infrastructure (SMTP on port 1025, UI on port 8025).

### Rate Limiting

Email sending is rate-limited via a token bucket at the provider level:
- Default: 2 sends/second (configurable via `EMAIL_RATE_LIMIT`)
- Test email domains (`@example.com`, `@test.com`, `@localhost`) are skipped automatically

## Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `-kafka-brokers` | `localhost:9092` | Kafka broker addresses |
| `-notifications-ready-topic` | `notifications.ready` | Input topic |
| `-consumer-group-id` | `sender-group` | Kafka consumer group |
| `-postgres-dsn` | `postgres://...` | Postgres connection string |

| Env Var | Default | Description |
|---------|---------|-------------|
| `SMTP_HOST` | `localhost` | SMTP server |
| `SMTP_PORT` | `1025` | SMTP port |
| `SMTP_USER` | - | SMTP username |
| `SMTP_PASSWORD` | - | SMTP password |
| `SMTP_FROM` | `alerts@alerting-platform.local` | From address |
| `EMAIL_RATE_LIMIT` | `2` | Sends per second |
| `EMAIL_PROVIDER` | auto | Force provider: `resend`, `ses`, or SMTP |

## Events

### Input: `notifications.ready`

```json
{
  "notification_id": "550e8400-...",
  "client_id": "client-456",
  "alert_id": "alert-123"
}
```

## Running

```bash
# From project root: start infrastructure first
make setup-infra && make run-migrations

# Then run this service
cd services/sender
make run-all
```

See `.env.example` for email configuration template.

## Testing

```bash
make test
```

## Key Properties

- **Idempotent**: Skips if notification is already `SENT`
- **Multi-channel**: Strategy pattern selects sender based on endpoint type
- **Rate-limited**: Token bucket prevents external API rate limit errors
- **At-least-once**: May re-send after crash (mitigated by status check + provider idempotency keys)
