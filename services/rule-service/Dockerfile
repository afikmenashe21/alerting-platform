# Multi-stage build for rule-service
# Stage 1: Build
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Copy go.mod files and pkg dependencies
COPY services/rule-service/go.mod services/rule-service/go.sum ./services/rule-service/
COPY pkg/ ./pkg/

# Download dependencies for rule-service
WORKDIR /build/services/rule-service
RUN go mod download

# Copy rule-service source code
COPY services/rule-service/ ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/rule-service ./cmd/rule-service

# Stage 2: Runtime
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/rule-service .

# Expose port (rule-service HTTP API)
EXPOSE 8080

# Run as non-root user
RUN adduser -D -u 10000 appuser
USER appuser

ENTRYPOINT ["/app/rule-service"]
