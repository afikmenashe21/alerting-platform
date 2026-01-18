#!/bin/bash
# Run all services in the alerting platform
# This script starts all services in separate background processes

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

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

echo_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

echo_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Services to run (in dependency order)
SERVICES=(
    "rule-service"
    "rule-updater"
    "evaluator"
    "aggregator"
    "sender"
    "alert-producer"
)

# Check if we should run in background or foreground
BACKGROUND="${BACKGROUND:-false}"
if [ "$1" = "--background" ] || [ "$1" = "-b" ]; then
    BACKGROUND=true
fi

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Running All Services${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# Step 1: Verify and start infrastructure if needed
echo_step "1/3 Verifying centralized infrastructure..."
# Temporarily disable set -e for this check
set +e
VERIFY_OUTPUT=$("$ROOT_DIR/scripts/infrastructure/verify-dependencies.sh" 2>&1)
VERIFY_EXIT_CODE=$?
set -e

if [ $VERIFY_EXIT_CODE -ne 0 ]; then
    echo_warn "Infrastructure is not running, starting it now..."
    "$ROOT_DIR/scripts/infrastructure/setup-infrastructure.sh" || {
        echo_error "Failed to start infrastructure"
        exit 1
    }
    # Wait a moment for services to stabilize
    sleep 3
    # Verify again after starting
    echo_info "Re-verifying infrastructure..."
    "$ROOT_DIR/scripts/infrastructure/verify-dependencies.sh" || {
        echo_error "Infrastructure started but verification failed"
        exit 1
    }
else
    # Show verification output if already running
    echo "$VERIFY_OUTPUT"
fi
echo ""

# Step 2: Run migrations (idempotent - safe to run multiple times)
echo_step "2/3 Running database migrations..."
if "$ROOT_DIR/scripts/migrations/run-migrations.sh"; then
    echo_success "Migrations are up to date"
else
    echo_error "Migration check failed"
    exit 1
fi
echo ""

# Step 3: Start services
echo_step "3/3 Starting all services..."
echo ""

# Create logs directory
LOGS_DIR="$ROOT_DIR/logs"
mkdir -p "$LOGS_DIR"

# Track PIDs
PIDS=()

# Function to cleanup on exit
cleanup() {
    echo ""
    echo_warn "Shutting down all services..."
    for pid in "${PIDS[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            echo_info "Stopping process $pid..."
            kill "$pid" 2>/dev/null || true
        fi
    done
    wait
    echo_success "All services stopped"
    exit 0
}

trap cleanup SIGINT SIGTERM

# Start each service
for service in "${SERVICES[@]}"; do
    SERVICE_DIR="$ROOT_DIR/services/$service"
    
    if [ ! -d "$SERVICE_DIR" ]; then
        echo_warn "Service directory not found: $service (skipping)"
        continue
    fi
    
    if [ ! -f "$SERVICE_DIR/scripts/run-all.sh" ]; then
        echo_warn "run-all.sh not found for $service (skipping)"
        continue
    fi
    
    echo_info "Starting $service..."
    
    if [ "$BACKGROUND" = "true" ]; then
        # Run in background, redirect output to log file
        LOG_FILE="$LOGS_DIR/${service}.log"
        "$SERVICE_DIR/scripts/run-all.sh" > "$LOG_FILE" 2>&1 &
        PID=$!
        PIDS+=($PID)
        echo_success "  $service started (PID: $PID, log: $LOG_FILE)"
    else
        # Run in foreground in a new terminal (if available)
        if command -v osascript &> /dev/null && [ "$(uname)" = "Darwin" ]; then
            # macOS - open new terminal window with service logs
            osascript -e "tell application \"Terminal\" to activate" -e "tell application \"Terminal\" to do script \"cd '$SERVICE_DIR' && echo '=== $service ===' && ./scripts/run-all.sh\"" &
            echo_success "  $service started in new terminal window"
            echo_info "    Check the new Terminal window for $service logs"
        elif command -v gnome-terminal &> /dev/null; then
            # Linux - gnome-terminal
            gnome-terminal --title="$service" -- bash -c "cd '$SERVICE_DIR' && echo '=== $service ===' && ./scripts/run-all.sh; exec bash" &
            echo_success "  $service started in new terminal window"
        else
            # Fallback: run in background with log file
            LOG_FILE="$LOGS_DIR/${service}.log"
            "$SERVICE_DIR/scripts/run-all.sh" > "$LOG_FILE" 2>&1 &
            PID=$!
            PIDS+=($PID)
            echo_success "  $service started (PID: $PID, log: $LOG_FILE)"
            echo_info "    View logs: tail -f $LOG_FILE"
        fi
    fi
    
    # Small delay between starting services
    sleep 2
done

echo ""
if [ "$BACKGROUND" = "true" ]; then
    echo_success "All services started in background"
    echo_info "View logs in: $LOGS_DIR/"
    echo_info "Stop all services with: pkill -f 'run-all.sh'"
    echo ""
    echo_info "Press Ctrl+C to stop all services"
    # Wait for all background processes
    wait
else
    echo_success "All services started"
    echo_info "Services are running in separate terminal windows"
    echo_info "Close the terminal windows to stop individual services"
    echo ""
    echo_info "To run in background mode: $0 --background"
fi
