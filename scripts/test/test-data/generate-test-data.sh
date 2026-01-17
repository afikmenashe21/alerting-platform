#!/bin/bash
# Generate test data: 100 clients with rules and endpoints

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Default DSN
DSN="${POSTGRES_DSN:-postgres://postgres:postgres@localhost:5432/alerting?sslmode=disable}"

# Allow override via command line
if [ $# -gt 0 ]; then
    DSN="$1"
fi

echo "Generating test data..."
echo "Database: $DSN"
echo ""

cd "$SCRIPT_DIR"

# Check if go is available
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed or not in PATH"
    exit 1
fi

# Download dependencies if needed
if [ ! -f "go.sum" ]; then
    echo "Downloading dependencies..."
    go mod download
    go mod tidy
fi

# Run the Go script
go run generate-test-data.go "$DSN"

echo ""
echo "Done! Test data generated successfully."
