# Multi-stage build for metrics-service
# Stage 1: Build
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Copy go.mod files and pkg dependencies
COPY services/metrics-service/go.mod services/metrics-service/go.sum ./services/metrics-service/
COPY pkg/ ./pkg/

# Download dependencies for metrics-service
WORKDIR /build/services/metrics-service
RUN go mod download

# Copy metrics-service source code
COPY services/metrics-service/ ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/metrics-service ./cmd/metrics-service

# Stage 2: Runtime
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/metrics-service .

# Expose port (metrics-service HTTP API)
EXPOSE 8083

# Run as non-root user
RUN adduser -D -u 10000 appuser
USER appuser

ENTRYPOINT ["/app/metrics-service"]
