# Multi-stage build for sender
# Stage 1: Build
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Copy go.mod files and pkg dependencies
COPY services/sender/go.mod services/sender/go.sum ./services/sender/
COPY pkg/ ./pkg/

# Download dependencies for sender
WORKDIR /build/services/sender
RUN go mod download

# Copy sender source code
COPY services/sender/ ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/sender ./cmd/sender

# Stage 2: Runtime
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/sender .

# Run as non-root user
RUN adduser -D -u 10000 appuser
USER appuser

ENTRYPOINT ["/app/sender"]
