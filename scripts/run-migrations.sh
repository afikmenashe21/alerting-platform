#!/bin/bash

# Centralized migration runner
# This script runs ALL migrations from ALL services in the correct order
# Services should NOT run migrations themselves - use this script instead

set -e

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

# Find migrate tool
find_migrate_tool() {
    if command -v migrate &> /dev/null; then
        echo "migrate"
        return 0
    elif [ -f ~/go/bin/migrate ]; then
        echo ~/go/bin/migrate
        return 0
    elif [ -n "$GOPATH" ] && [ -f "$GOPATH/bin/migrate" ]; then
        echo "$GOPATH/bin/migrate"
        return 0
    fi
    return 1
}

# Postgres connection string
POSTGRES_DSN="${POSTGRES_DSN:-postgres://postgres:postgres@127.0.0.1:5432/alerting?sslmode=disable}"
POSTGRES_CONTAINER="alerting-platform-postgres"

# Check if migrate tool exists
MIGRATE=$(find_migrate_tool)
if [ -z "$MIGRATE" ]; then
    echo_error "migrate tool not found"
    echo_error "Install with: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
    exit 1
fi

echo_info "Using migrate tool: $MIGRATE"

# Verify Postgres is accessible
if ! docker ps --format "{{.Names}}" | grep -q "^${POSTGRES_CONTAINER}$"; then
    echo_error "Postgres container '$POSTGRES_CONTAINER' is not running"
    echo_error "Start it with: docker compose up -d postgres"
    exit 1
fi

# Ensure database exists
echo_info "Ensuring database 'alerting' exists..."
DB_EXISTS=$(docker exec "$POSTGRES_CONTAINER" psql -U postgres -tAc "SELECT 1 FROM pg_database WHERE datname='alerting';" 2>/dev/null || echo "")
if [ -z "$DB_EXISTS" ]; then
    echo_info "Creating database 'alerting'..."
    docker exec "$POSTGRES_CONTAINER" psql -U postgres -c "CREATE DATABASE alerting;" 2>/dev/null || {
        echo_error "Failed to create database 'alerting'"
        exit 1
    }
    echo_success "Database 'alerting' created"
else
    echo_success "Database 'alerting' exists"
fi

# Collect all migration files from all services
echo_info "Collecting migrations from all services..."
TEMP_MIGRATIONS=$(mktemp -d)
trap "rm -rf $TEMP_MIGRATIONS" EXIT

# Find all migration files and copy them to temp directory with proper naming
# Search in services/*/migrations/ directories
find . -path "*/services/*/migrations/*.up.sql" -type f | sort | while read upfile; do
    version=$(basename "$upfile" | sed 's/^\([0-9]*\)_.*/\1/')
    service=$(echo "$upfile" | sed 's|^\./\([^/]*\)/.*|\1|')
    basename=$(basename "$upfile" | sed 's/^[0-9]*_//')
    
    # Create properly numbered filename (pad version to 6 digits for sorting)
    newname=$(printf "%06d_%s" "$version" "$basename")
    cp "$upfile" "$TEMP_MIGRATIONS/$newname"
    
    # Copy down migration if it exists
    downfile=$(echo "$upfile" | sed 's/\.up\.sql$/.down.sql/')
    if [ -f "$downfile" ]; then
        downbasename=$(basename "$downfile" | sed 's/^[0-9]*_//')
        downnewname=$(printf "%06d_%s" "$version" "$downbasename")
        cp "$downfile" "$TEMP_MIGRATIONS/$downnewname"
    fi
done

cd "$TEMP_MIGRATIONS"
MIGRATION_COUNT=$(ls -1 *.up.sql 2>/dev/null | wc -l | tr -d ' ')
if [ "$MIGRATION_COUNT" -eq 0 ]; then
    echo_error "No migration files found"
    exit 1
fi
echo_info "Found $MIGRATION_COUNT migration(s)"

# Check current database version
echo_info "Checking current database version..."
CURRENT_VERSION=$(docker exec "$POSTGRES_CONTAINER" psql -U postgres -d alerting -tAc "SELECT version FROM schema_migrations;" 2>/dev/null || echo "")
if [ -z "$CURRENT_VERSION" ]; then
    echo_info "No migrations applied yet (fresh database)"
    CURRENT_VERSION="0"
else
    echo_info "Current database version: $CURRENT_VERSION"
fi

# Check for dirty state
DIRTY=$(docker exec "$POSTGRES_CONTAINER" psql -U postgres -d alerting -tAc "SELECT dirty FROM schema_migrations;" 2>/dev/null || echo "false")
if [ "$DIRTY" = "t" ]; then
    echo_warn "Database is in dirty state - previous migration may have failed"
    echo_warn "You may need to manually fix this. See MIGRATION_STRATEGY.md for troubleshooting"
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Run migrations
echo_info "Running migrations..."
set +e
MIGRATE_OUTPUT=$($MIGRATE -path "$TEMP_MIGRATIONS" -database "$POSTGRES_DSN" up 2>&1)
MIGRATE_EXIT=$?
set -e

if [ $MIGRATE_EXIT -ne 0 ]; then
    echo_error "Migration failed:"
    echo "$MIGRATE_OUTPUT" | while IFS= read -r line; do
        echo_error "  $line"
    done
    exit 1
fi

# Show migration output
if [ -n "$MIGRATE_OUTPUT" ]; then
    echo "$MIGRATE_OUTPUT" | while IFS= read -r line; do
        if echo "$line" | grep -qi "error\|failed"; then
            echo_error "  $line"
        else
            echo_info "  $line"
        fi
    done
fi

# Verify final version
FINAL_VERSION=$(docker exec "$POSTGRES_CONTAINER" psql -U postgres -d alerting -tAc "SELECT version FROM schema_migrations;" 2>/dev/null || echo "")
echo_success "Migrations completed successfully"
echo_info "Final database version: $FINAL_VERSION"

# Verify critical tables exist
echo_info "Verifying critical tables..."
TABLES=("clients" "rules" "endpoints" "notifications")
ALL_EXIST=true
for table in "${TABLES[@]}"; do
    EXISTS=$(docker exec "$POSTGRES_CONTAINER" psql -U postgres -d alerting -tAc "SELECT EXISTS(SELECT FROM information_schema.tables WHERE table_name = '$table');" 2>/dev/null || echo "false")
    if [ "$EXISTS" = "t" ]; then
        echo_success "  Table '$table' exists"
    else
        echo_warn "  Table '$table' does not exist (may not be created yet)"
        ALL_EXIST=false
    fi
done

if [ "$ALL_EXIST" = true ]; then
    echo_success "All critical tables verified"
fi
