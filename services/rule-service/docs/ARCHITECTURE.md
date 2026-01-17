# Rule-Service Architecture

This document describes the architecture and design patterns used in the rule-service.

## Overview

The rule-service is a control-plane API service that provides REST endpoints for managing clients, rules, and endpoints. It follows a modular architecture with clear separation of concerns.

## Architecture Pattern

### Modular Design with Router Pattern

The service uses a **Router Pattern** for HTTP routing and a **Handler Pattern** for request processing:

```
cmd/rule-service/main.go
├── Initialization (config, database, producer)
├── Router setup
└── HTTP server startup

internal/router/
└── router.go          # Route configuration and middleware (CORS)

internal/handlers/
├── base.go            # Base Handlers struct and dependencies
└── handlers.go        # HTTP handler implementations (by resource)

internal/database/
└── database.go        # Data access layer

internal/producer/
└── producer.go        # Kafka event publishing
```

## Directory Structure

```
rule-service/
├── cmd/
│   └── rule-service/
│       └── main.go              # CLI entry point, initialization
├── internal/
│   ├── router/                  # HTTP routing
│   │   └── router.go           # Route configuration and middleware
│   ├── handlers/                # HTTP handlers
│   │   ├── base.go             # Base Handlers struct
│   │   └── handlers.go         # Handler implementations
│   ├── database/               # Data access layer
│   │   └── database.go
│   ├── producer/               # Kafka event publishing
│   │   └── producer.go
│   ├── events/                 # Event definitions
│   │   └── events.go
│   └── config/                 # Configuration
│       └── config.go
├── migrations/                  # Database migrations
├── scripts/                     # Service scripts
│   └── run-all.sh
├── memory-bank/                 # Service memory bank
├── Makefile
└── README.md
```

## Components

### Router (`internal/router/`)

The router package provides HTTP route configuration and middleware:

- **Route Setup**: Configures all HTTP endpoints
- **CORS Middleware**: Handles cross-origin requests
- **Server Creation**: Creates HTTP server with timeouts

**Key Features:**
- Centralized route configuration
- CORS middleware for UI integration
- Clean separation from handler logic

### Handlers (`internal/handlers/`)

The handlers package contains HTTP request handlers organized by resource:

- **Base Handlers**: Common struct with database and producer dependencies
- **Resource Handlers**: Separate handlers for clients, rules, endpoints, notifications

**Key Features:**
- Resource-based organization
- Shared dependencies via base struct
- Consistent error handling

### Database (`internal/database/`)

The database package provides data access operations:

- **CRUD Operations**: Create, read, update, delete for all entities
- **Optimistic Locking**: Version-based locking for rule updates
- **Transaction Management**: Proper transaction handling

### Producer (`internal/producer/`)

The producer package handles Kafka event publishing:

- **Event Publishing**: Publishes `rule.changed` events
- **Partitioning**: Keys events by `rule_id` for tenant locality
- **Error Handling**: Graceful error handling with logging

## Design Patterns

### Router Pattern

The router pattern separates route configuration from handler logic:

```go
// Router sets up routes
router := router.NewRouter(handlers)

// Handlers implement business logic
handlers.CreateClient(w, r)
```

**Benefits:**
- Easy to add new routes
- Centralized middleware application
- Testable route configuration

### Handler Pattern

Handlers are organized by resource type with shared dependencies:

```go
type Handlers struct {
    db       *database.DB
    producer *producer.Producer
}
```

**Benefits:**
- Clear separation of concerns
- Easy to test handlers independently
- Consistent dependency injection

## Event Flow

1. **HTTP Request** → Router → Handler
2. **Handler** → Database (CRUD operation)
3. **Handler** → Producer (publish `rule.changed` event)
4. **Response** → Client

## Error Handling

- **Validation Errors**: Return 400 Bad Request
- **Not Found**: Return 404 Not Found
- **Conflict**: Return 409 Conflict (optimistic locking)
- **Server Errors**: Return 500 Internal Server Error with logging

## Extensibility

### Adding New Routes

1. Add route in `internal/router/router.go`
2. Implement handler in `internal/handlers/handlers.go`
3. Add database methods if needed

### Adding New Resources

1. Create database methods in `internal/database/database.go`
2. Add handlers in `internal/handlers/handlers.go`
3. Add routes in `internal/router/router.go`
4. Add migrations if needed

## Testing

The modular architecture makes testing easier:

- **Router Tests**: Test route configuration
- **Handler Tests**: Mock database and producer
- **Database Tests**: Test data access logic
- **Integration Tests**: Test full request flow
