# Multi-stage build for aggregator
# Stage 1: Build
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Copy go.mod files and pkg dependencies
COPY services/aggregator/go.mod services/aggregator/go.sum ./services/aggregator/
COPY pkg/ ./pkg/

# Download dependencies for aggregator
WORKDIR /build/services/aggregator
RUN go mod download

# Copy aggregator source code
COPY services/aggregator/ ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/aggregator ./cmd/aggregator

# Stage 2: Runtime
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/aggregator .

# Run as non-root user
RUN adduser -D -u 10000 appuser
USER appuser

ENTRYPOINT ["/app/aggregator"]
