# Sender Service

The sender service consumes `notifications.ready` events from Kafka, fetches notifications and email endpoints from the database, sends email notifications (stub implementation for MVP), and updates notification status to SENT.

## Overview

The sender service is the final step in the alerting platform pipeline:

1. **Consumes** `notifications.ready` events from Kafka
2. **Fetches** notification details from Postgres
3. **Queries** email endpoints for matching rule IDs
4. **Sends** email notifications (stub: logs for MVP)
5. **Updates** notification status to SENT

## Features

- ✅ Kafka consumer with at-least-once delivery semantics
- ✅ Idempotent processing (skips if already SENT)
- ✅ Multi-channel sending (email, Slack, webhooks)
- ✅ Strategy pattern for extensible sender types
- ✅ Graceful shutdown handling
- ✅ Structured logging

## Documentation

All documentation is available in the [`docs/`](docs/) directory:

- **[Documentation Index](docs/README.md)** - Complete documentation index
- **[Architecture](docs/ARCHITECTURE.md)** - Service architecture and design patterns
- **[Gmail SMTP Setup](docs/GMAIL_SETUP.md)** - Complete guide for configuring Gmail SMTP
- **[Troubleshooting](docs/TROUBLESHOOTING.md)** - Common issues and solutions for email sending
- **[Testing Guide](docs/TESTING.md)** - Testing strategies and test coverage

## Quick Start

### Prerequisites

- Go 1.22+
- Docker and Docker Compose
- Postgres database with `notifications` and `endpoints` tables
- Kafka with `notifications.ready` topic

### One-Command Setup and Run

The easiest way to get started:

```bash
make run-all
```

This will:
1. ✅ Verify Go 1.22+ is installed
2. ✅ Check Docker is installed and running
3. ✅ Download Go dependencies
4. ✅ Start Postgres, Kafka, and Zookeeper
5. ✅ Wait for services to be ready
6. ✅ Create Kafka topics
7. ✅ Build and run the sender service

### Manual Setup

```bash
# Download dependencies
make deps

# Start infrastructure
make setup

# Build
make build

# Run
make run
```

## Configuration

The service accepts the following command-line flags:

- `-kafka-brokers`: Kafka broker addresses (default: `localhost:9092`)
- `-notifications-ready-topic`: Kafka topic name (default: `notifications.ready`)
- `-consumer-group-id`: Kafka consumer group ID (default: `sender-group`)
- `-postgres-dsn`: PostgreSQL connection string (default: `postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable`)

### Email Configuration (Environment Variables)

The email sender can be configured via environment variables:

- `SMTP_HOST`: SMTP server hostname (default: `localhost`)
- `SMTP_PORT`: SMTP server port (default: `1025`)
- `SMTP_USER`: SMTP username (optional, required for authenticated SMTP)
- `SMTP_PASSWORD`: SMTP password (optional, required for authenticated SMTP)
- `SMTP_FROM`: Email address to send from (default: `alerts@alerting-platform.local`)

**Configuration Options:**
- Use environment variables directly: `export SMTP_HOST=...`
- Use a `.env` file: Copy `.env.example` to `.env` and fill in your credentials
- Use a secrets manager in production environments

See `.env.example` in the sender directory for a template.

#### Gmail Configuration

For detailed Gmail SMTP setup instructions, see **[Gmail SMTP Setup Guide](docs/GMAIL_SETUP.md)**.

Quick setup:
```bash
export SMTP_HOST=smtp.gmail.com
export SMTP_PORT=587
export SMTP_USER=your-email@gmail.com
export SMTP_PASSWORD=your-app-password
export SMTP_FROM=your-email@gmail.com
```

**Important**: 
- Replace `your-email@gmail.com` with your actual Gmail address
- Replace `your-app-password` with a Gmail App Password (required if 2FA is enabled)
- Never commit credentials to version control - use environment variables or a `.env` file

#### Local Testing with MailHog

For local testing, use MailHog (included in infrastructure):

```bash
export SMTP_HOST=localhost
export SMTP_PORT=1025
export SMTP_FROM=alerts@alerting-platform.local
# SMTP_USER and SMTP_PASSWORD not needed for MailHog
```

View captured emails at: http://localhost:8025

## Architecture

### Processing Flow

```
Kafka (notifications.ready)
    ↓
Consumer reads event
    ↓
Fetch notification from DB
    ↓
Check if already SENT (idempotency)
    ↓
Query email endpoints for rule_ids
    ↓
Send email (stub: log)
    ↓
Update status to SENT
    ↓
Commit Kafka offset
```

### Idempotency

The service implements idempotency at multiple levels:

1. **Status Check**: Before sending, checks if notification is already SENT
2. **Status Update**: After successful send, updates status atomically
3. **Safe Retry**: If status update fails, retry will skip (already sent check)

This ensures at-least-once delivery semantics: if the service crashes after sending but before updating status, Kafka will redeliver, but the service will skip (already sent).

### Email Sender

The email sender is currently a stub implementation that logs email details. It's designed to be extensible:

- `Sender` struct can hold multiple sender implementations
- `SendNotification` aggregates endpoints from all matching rules
- Collects unique email addresses to avoid duplicates

Future enhancements:
- Real email service integration (SendGrid, SES, etc.)
- Slack sender
- Webhook sender

## Database Schema

The service queries two tables:

### notifications (data-plane)
- `notification_id` (PK)
- `client_id`, `alert_id`
- `severity`, `source`, `name`
- `context` (JSONB)
- `rule_ids` (TEXT[])
- `status` (RECEIVED, SENT)

### endpoints (control-plane)
- `endpoint_id` (PK)
- `rule_id` (FK)
- `type` (email, webhook, slack)
- `value` (email address, URL, etc.)
- `enabled` (boolean)

## Makefile Targets

- `make build` - Build the binary
- `make run` - Run the service
- `make test` - Run tests
- `make clean` - Remove build artifacts
- `make deps` - Download dependencies
- `make run-all` - Setup and run (recommended)
- `make setup` - Start infrastructure and create topics
- `make up` - Start Docker services
- `make down` - Stop Docker services
- `make logs` - View Docker logs
- `make db-query` - Query recent notifications
- `make db-count` - Count notifications by status
- `make db-sent` - Show SENT notifications

## Development

### Project Structure

```
sender/
├── cmd/
│   └── sender/
│       └── main.go          # Main entry point
├── internal/
│   ├── config/
│   │   └── config.go        # Configuration
│   ├── consumer/
│   │   └── consumer.go      # Kafka consumer
│   ├── database/
│   │   └── database.go      # Database operations
│   ├── events/
│   │   └── events.go        # Event structures
│   └── sender/
│       ├── sender.go        # Multi-channel sender coordinator
│       ├── email/           # Email sender implementation
│       ├── slack/           # Slack sender implementation
│       ├── webhook/         # Webhook sender implementation
│       ├── strategy/        # Strategy pattern for senders
│       └── payload/         # Payload builders
├── docs/                    # 📚 All documentation
│   ├── README.md           # Documentation index
│   ├── ARCHITECTURE.md     # Architecture and design patterns
│   ├── GMAIL_SETUP.md      # Gmail SMTP configuration guide
│   ├── TROUBLESHOOTING.md  # Troubleshooting guide
│   └── TESTING.md          # Testing guide
├── scripts/
│   └── run-all.sh           # Setup and run script
├── memory-bank/             # Service memory bank (design decisions)
├── .env.example             # Environment variable template
├── Makefile                 # Build and run targets
└── go.mod                   # Go dependencies
```

### Adding New Sender Types

To add a new sender type (e.g., Slack):

1. Add sender implementation in `internal/sender/`:
   ```go
   func (s *Sender) SendSlack(ctx context.Context, webhookURL string, notification *database.Notification) error {
       // Implementation
   }
   ```

2. Update `SendNotification` to route to appropriate sender based on endpoint type

3. Query endpoints with the new type in `GetEmailEndpointsByRuleIDs` or create a new method

## Testing

See **[Testing Guide](docs/TESTING.md)** for detailed testing information.

```bash
# Run tests
make test

# Check database
make db-query
make db-count
make db-sent
```

## Troubleshooting

For detailed troubleshooting information, see **[Troubleshooting Guide](docs/TROUBLESHOOTING.md)**.

### Quick Troubleshooting

**Service won't start:**
- Check Docker is running: `docker ps`
- Check Postgres is ready: `docker exec alerting-platform-postgres pg_isready -U postgres`
- Check Kafka is ready: `docker exec alerting-platform-kafka kafka-broker-api-versions --bootstrap-server localhost:9092`

**No notifications being processed:**
- Check Kafka topic exists: `docker exec alerting-platform-kafka kafka-topics --list --bootstrap-server localhost:9092`
- Check notifications exist: `make db-query`
- Check notifications have RECEIVED status: `make db-count`
- Check service logs for errors

**Emails not being sent:**
- Check email endpoints exist in `endpoints` table
- Check endpoints are enabled (`enabled = TRUE`)
- Check endpoints have type `email`
- Check rule_ids in notification match rule_ids with endpoints
- See [Troubleshooting Guide](docs/TROUBLESHOOTING.md) for email-specific issues

## License

This is an alerting platform project for learning system design and Go implementation.
