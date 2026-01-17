#!/bin/bash
# Run a single service in a separate terminal
# Usage: ./scripts/run-single-service.sh <service-name>

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$ROOT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

echo_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

echo_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check service name argument
if [ -z "$1" ]; then
    echo_error "Usage: $0 <service-name>"
    echo_error "Example: $0 alert-producer"
    exit 1
fi

SERVICE="$1"
SERVICE_DIR="$ROOT_DIR/services/$SERVICE"

# Verify service directory exists
if [ ! -d "$SERVICE_DIR" ]; then
    echo_error "Service directory not found: $SERVICE"
    exit 1
fi

# Verify run-all.sh exists
if [ ! -f "$SERVICE_DIR/scripts/run-all.sh" ]; then
    echo_error "run-all.sh not found for $SERVICE"
    exit 1
fi

# Verify infrastructure is running
echo_info "Verifying infrastructure..."
if ! "$ROOT_DIR/scripts/verify-dependencies.sh" > /dev/null 2>&1; then
    echo_error "Infrastructure is not running"
    echo_error "Start infrastructure with: make setup-infra"
    exit 1
fi

# Start service in a new terminal
echo_info "Starting $SERVICE in a new terminal window..."

if command -v osascript &> /dev/null && [ "$(uname)" = "Darwin" ]; then
    # macOS - open new terminal window
    osascript -e "tell application \"Terminal\" to activate" -e "tell application \"Terminal\" to do script \"cd '$SERVICE_DIR' && echo '=== $SERVICE ===' && ./scripts/run-all.sh\"" &
    echo_success "$SERVICE started in new terminal window"
    echo_info "Check the new Terminal window for $SERVICE logs"
elif command -v gnome-terminal &> /dev/null; then
    # Linux - gnome-terminal
    gnome-terminal --title="$SERVICE" -- bash -c "cd '$SERVICE_DIR' && echo '=== $SERVICE ===' && ./scripts/run-all.sh; exec bash" &
    echo_success "$SERVICE started in new terminal window"
else
    # Fallback: run in current terminal
    echo_info "Running $SERVICE in current terminal (no separate terminal available)"
    cd "$SERVICE_DIR"
    ./scripts/run-all.sh
fi
