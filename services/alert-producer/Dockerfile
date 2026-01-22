# Multi-stage build for alert-producer-api
# Stage 1: Build
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Copy go.mod files and pkg dependencies
COPY services/alert-producer/go.mod services/alert-producer/go.sum ./services/alert-producer/
COPY pkg/ ./pkg/

# Download dependencies for alert-producer
WORKDIR /build/services/alert-producer
RUN go mod download

# Copy alert-producer source code
COPY services/alert-producer/ ./

# Build the API binary (for production use)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/alert-producer-api ./cmd/alert-producer-api

# Stage 2: Runtime
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/alert-producer-api .

# Expose port (alert-producer-api HTTP API)
EXPOSE 8081

# Run as non-root user
RUN adduser -D -u 10000 appuser
USER appuser

ENTRYPOINT ["/app/alert-producer-api"]
