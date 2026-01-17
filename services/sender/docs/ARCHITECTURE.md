# Sender Architecture

This document describes the architecture and design patterns used in the sender service.

## Overview

The sender service is the final step in the alerting platform pipeline. It consumes `notifications.ready` events from Kafka, fetches notifications and endpoints from the database, and sends notifications via multiple channels (email, Slack, webhooks).

## Architecture Pattern

### Modular Design with Strategy Pattern

The service uses a **Strategy Pattern** for multi-channel notification sending:

```
cmd/sender/main.go
├── Initialization (config, database, consumer, sender coordinator)
└── Processing loop

internal/sender/
├── sender.go              # Coordinator for multi-channel sending
├── strategy/              # Strategy interface and registry
│   └── strategy.go
├── email/                 # Email sender implementation
│   └── email.go
├── slack/                 # Slack sender implementation
│   └── slack.go
├── webhook/               # Webhook sender implementation
│   └── webhook.go
└── payload/               # Payload builders
    └── payload.go
```

## Directory Structure

```
sender/
├── cmd/
│   └── sender/
│       └── main.go              # CLI entry point, initialization
├── internal/
│   ├── sender/                  # Sender coordination
│   │   ├── sender.go           # Main coordinator
│   │   ├── strategy/           # Strategy pattern
│   │   │   └── strategy.go
│   │   ├── email/              # Email sender
│   │   │   └── email.go
│   │   ├── slack/              # Slack sender
│   │   │   └── slack.go
│   │   ├── webhook/            # Webhook sender
│   │   │   └── webhook.go
│   │   └── payload/            # Payload builders
│   │       └── payload.go
│   ├── consumer/               # Kafka consumer
│   │   └── consumer.go
│   ├── database/               # Data access layer
│   │   └── database.go
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

### Sender Coordinator (`internal/sender/sender.go`)

The sender coordinator routes notifications to appropriate strategies:

- **SendNotification**: Routes to strategies based on endpoint type
- **GroupEndpoints**: Groups endpoints by type and value

**Key Features:**
- Multi-channel routing
- Endpoint grouping (deduplication)
- Partial failure handling

### Strategy Pattern (`internal/sender/strategy/`)

The strategy package defines the sender interface and registry:

- **NotificationSender**: Interface for all sender types
- **Registry**: Manages sender registration and retrieval

**Key Features:**
- Extensible design
- Easy to add new sender types
- Type-safe sender retrieval

### Email Sender (`internal/sender/email/`)

The email sender implements SMTP email delivery:

- **Send**: Sends email via SMTP
- **BuildMessage**: Builds RFC 822 formatted messages

**Key Features:**
- SMTP protocol support
- Configurable server settings
- Multiple recipient support

### Slack Sender (`internal/sender/slack/`)

The Slack sender implements Incoming Webhooks API:

- **Send**: Sends formatted messages to Slack
- **BuildPayload**: Builds Slack message payload

**Key Features:**
- Incoming Webhooks API
- Color-coded attachments
- Severity-based formatting

### Webhook Sender (`internal/sender/webhook/`)

The webhook sender implements HTTP POST delivery:

- **Send**: Sends HTTP POST requests
- **BuildPayload**: Builds JSON payload

**Key Features:**
- HTTP POST with JSON
- Configurable timeout
- Full notification details

## Design Patterns

### Strategy Pattern

The strategy pattern enables multi-channel notification sending:

```go
// Register strategies
registry.Register(email.NewSender())
registry.Register(slack.NewSender())
registry.Register(webhook.NewSender())

// Route to appropriate strategy
sender := registry.Get(endpointType)
sender.Send(ctx, endpointValue, notification)
```

**Benefits:**
- Easy to add new sender types
- Clean separation of sender implementations
- Type-safe routing

### Coordinator Pattern

The coordinator pattern groups and routes notifications:

```go
// Coordinator groups endpoints and routes to strategies
sender.SendNotification(ctx, notification, endpoints)
```

**Benefits:**
- Centralized routing logic
- Endpoint deduplication
- Partial failure handling

## Processing Flow

1. **Read Event**: Read `notifications.ready` message from Kafka
2. **Fetch Notification**: Get notification details from database
3. **Idempotency Check**: Skip if already SENT
4. **Fetch Endpoints**: Get all endpoints for matching rule IDs
5. **Group by Type**: Group endpoints by type (email, slack, webhook)
6. **Route to Strategies**: Send via appropriate sender for each type
7. **Update Status**: Update notification status to SENT
8. **Commit Offset**: Commit Kafka offset after successful operations

## Multi-Channel Sending

The service supports multiple notification channels:

- **Email**: SMTP protocol (default: localhost:1025 for MailHog)
- **Slack**: Incoming Webhooks API
- **Webhook**: HTTP POST with JSON payload

**Features:**
- Groups endpoints by type to avoid duplicates
- Partial failures are logged but don't fail operation
- At least one successful send marks operation as successful

## Error Handling

- **Database Errors**: Log and skip, Kafka will redeliver
- **Send Errors**: Log but continue with other channels
- **Status Update Errors**: Log and skip, Kafka will redeliver
- **Partial Failures**: Log warnings but don't fail operation

## Extensibility

### Adding New Sender Types

1. Implement `NotificationSender` interface
2. Register in `sender.NewSender()`
3. Add endpoint type validation

**Example:**
```go
type SMSSender struct { ... }

func (s *SMSSender) Send(ctx context.Context, endpoint string, notification *database.Notification) error {
    // Implementation
}

// Register
registry.Register(NewSMSSender())
```

## Testing

The modular architecture makes testing easier:

- **Sender Tests**: Mock strategies and test routing
- **Strategy Tests**: Test individual sender implementations
- **Integration Tests**: Test full notification sending flow
