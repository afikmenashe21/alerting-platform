# Multi-stage build for rule-updater
# Stage 1: Build
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Copy go.mod files and pkg dependencies
COPY services/rule-updater/go.mod services/rule-updater/go.sum ./services/rule-updater/
COPY pkg/ ./pkg/

# Download dependencies for rule-updater
WORKDIR /build/services/rule-updater
RUN go mod download

# Copy rule-updater source code
COPY services/rule-updater/ ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/rule-updater ./cmd/rule-updater

# Stage 2: Runtime
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/rule-updater .

# Run as non-root user
RUN adduser -D -u 10000 appuser
USER appuser

ENTRYPOINT ["/app/rule-updater"]
