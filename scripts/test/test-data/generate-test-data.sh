#!/bin/bash
# Generate test data: 1,500 clients × 300 rules × 2 endpoints = 450k rules, 900k endpoints
# Targets 90% of evaluator memory capacity

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# Default DSN (override with POSTGRES_DSN env var or first argument)
DSN="${POSTGRES_DSN:-postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable}"

if [ $# -gt 0 ]; then
    DSN="$1"
fi

echo "=== Alerting Platform Test Data Generator ==="
echo ""
echo "Target: 1,500 clients | 450,000 rules | 900,000 endpoints"
echo "Database: ${DSN%%@*}@***"
echo ""

cd "$SCRIPT_DIR"

if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed or not in PATH"
    exit 1
fi

# Download dependencies if needed
if [ ! -f "go.sum" ] || [ "go.mod" -nt "go.sum" ]; then
    echo "Downloading dependencies..."
    go mod download
    go mod tidy
fi

# Run the Go script
go run generate-test-data.go "$DSN"

echo ""
echo "Done! Test data generated successfully."
