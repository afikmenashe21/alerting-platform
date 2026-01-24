# Multi-stage build for evaluator
# Stage 1: Build
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Copy go.mod files and pkg dependencies
COPY services/evaluator/go.mod services/evaluator/go.sum ./services/evaluator/
COPY pkg/ ./pkg/

# Download dependencies for evaluator
WORKDIR /build/services/evaluator
RUN go mod download

# Copy evaluator source code
COPY services/evaluator/ ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/evaluator ./cmd/evaluator

# Stage 2: Runtime
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/evaluator .

# Run as non-root user
RUN adduser -D -u 10000 appuser
USER appuser

ENTRYPOINT ["/app/evaluator"]
