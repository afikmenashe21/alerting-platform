#!/bin/bash
# Stop all application services
# This script stops all running service processes

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

echo_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

echo_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}  Stopping All Services${NC}"
echo -e "${YELLOW}========================================${NC}"
echo ""

# Services to stop
SERVICES=(
    "rule-service"
    "rule-updater"
    "evaluator"
    "aggregator"
    "sender"
    "alert-producer"
)

STOPPED_COUNT=0

# Method 1: Kill processes by name pattern (for background services)
echo_info "Stopping background service processes..."
if pkill -f 'run-all.sh' 2>/dev/null; then
    echo_success "Stopped background service processes"
    STOPPED_COUNT=$((STOPPED_COUNT + 1))
else
    echo_info "No background service processes found"
fi

# Method 2: Kill individual service binaries
for service in "${SERVICES[@]}"; do
    # Try to find and kill the service binary
    SERVICE_BINARY=$(basename "$service")
    if pgrep -f "$SERVICE_BINARY" > /dev/null 2>&1; then
        echo_info "Stopping $service processes..."
        pkill -f "$SERVICE_BINARY" 2>/dev/null && {
            echo_success "  Stopped $service"
            STOPPED_COUNT=$((STOPPED_COUNT + 1))
        }
    fi
done

# Method 3: Try to close Terminal windows on macOS (if services were started in new terminals)
if command -v osascript &> /dev/null; then
    echo_info "Checking for service terminal windows (macOS)..."
    # Note: This is a best-effort attempt. Terminal windows opened by osascript
    # may not be easily identifiable, so we rely on process killing above.
    echo_info "  Terminal windows should be closed manually if services were started in new windows"
fi

# Method 4: Kill Go processes that might be running services
echo_info "Checking for Go service processes..."
GO_PROCESSES=$(pgrep -f "go run.*cmd" 2>/dev/null || true)
if [ -n "$GO_PROCESSES" ]; then
    echo_info "Found Go processes, stopping them..."
    pkill -f "go run.*cmd" 2>/dev/null && {
        echo_success "Stopped Go service processes"
        STOPPED_COUNT=$((STOPPED_COUNT + 1))
    }
fi

# Wait a moment for processes to terminate
sleep 1

# Check for any remaining service processes
REMAINING=$(pgrep -f "run-all.sh|go run.*cmd" 2>/dev/null || true)
if [ -n "$REMAINING" ]; then
    echo_warn "Some service processes may still be running:"
    pgrep -f "run-all.sh|go run.*cmd" | while read pid; do
        echo_info "  PID $pid: $(ps -p $pid -o comm= 2>/dev/null || echo 'unknown')"
    done
    echo_warn "You may need to kill them manually: kill <PID>"
else
    echo_success "All service processes stopped"
fi

echo ""
if [ $STOPPED_COUNT -gt 0 ]; then
    echo_success "Stopped $STOPPED_COUNT service(s)"
else
    echo_info "No running services found"
fi
echo ""
echo_info "Note: If services were started in separate terminal windows, close those windows manually"
echo_info "Note: Infrastructure (Docker containers) is still running. Use 'make stop-infra' to stop it"
