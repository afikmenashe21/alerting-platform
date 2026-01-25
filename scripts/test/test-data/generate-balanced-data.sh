#!/bin/bash
# Generate balanced test data for the alerting platform
# This script cleans ALL data and generates fresh test data
#
# Usage:
#   ./generate-balanced-data.sh                    # Uses production RDS
#   ./generate-balanced-data.sh local              # Uses localhost
#   POSTGRES_DSN="..." ./generate-balanced-data.sh # Custom DSN

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
echo_success() { echo -e "${GREEN}[✓]${NC} $1"; }
echo_error() { echo -e "${RED}[ERROR]${NC} $1"; }
echo_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }

cd "$(dirname "$0")"

echo -e "${GREEN}============================================${NC}"
echo -e "${GREEN}  Balanced Test Data Generator${NC}"
echo -e "${GREEN}============================================${NC}"
echo ""

# Determine database connection
if [ "$1" = "local" ]; then
    DB_HOST="${DB_HOST:-localhost}"
    DB_PORT="${DB_PORT:-5432}"
    DB_NAME="${DB_NAME:-alerting}"
    DB_USER="${DB_USER:-postgres}"
    DB_PASSWORD="${DB_PASSWORD:-postgres}"
    echo_info "Using local database"
elif [ -n "$POSTGRES_DSN" ]; then
    echo_info "Using custom POSTGRES_DSN"
    # Parse DSN for psql
    export PGPASSWORD=$(echo "$POSTGRES_DSN" | sed -n 's/.*:\/\/[^:]*:\([^@]*\)@.*/\1/p')
    DB_HOST=$(echo "$POSTGRES_DSN" | sed -n 's/.*@\([^:\/]*\).*/\1/p')
    DB_PORT=$(echo "$POSTGRES_DSN" | sed -n 's/.*:\([0-9]*\)\/.*/\1/p')
    DB_NAME=$(echo "$POSTGRES_DSN" | sed -n 's/.*\/\([^?]*\).*/\1/p')
    DB_USER=$(echo "$POSTGRES_DSN" | sed -n 's/.*:\/\/\([^:]*\):.*/\1/p')
else
    # Get production RDS connection from terraform
    echo_info "Getting production RDS connection from Terraform..."
    cd ../../../terraform
    RDS_ENDPOINT=$(terraform output -raw rds_endpoint 2>/dev/null || echo "")
    if [ -z "$RDS_ENDPOINT" ]; then
        echo_error "Failed to get RDS endpoint from Terraform. Use 'local' or set POSTGRES_DSN"
        exit 1
    fi
    DB_HOST=$(echo "$RDS_ENDPOINT" | cut -d: -f1)
    DB_PORT=$(echo "$RDS_ENDPOINT" | cut -d: -f2)
    DB_NAME="alerting"
    DB_USER="postgres"
    DB_PASSWORD=$(grep '^db_password' terraform.tfvars | sed 's/.*"\(.*\)".*/\1/')
    cd - > /dev/null
    echo_info "Using production RDS"
fi

export PGPASSWORD="$DB_PASSWORD"

echo_info "Host: $DB_HOST"
echo_info "Port: $DB_PORT"
echo_info "Database: $DB_NAME"
echo ""

echo_warn "This will DELETE ALL DATA including notifications!"
read -p "Are you sure you want to continue? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo_info "Aborted"
    exit 0
fi

echo ""
echo_info "Running data generation..."
echo ""

psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" < generate-balanced-data.sql

echo ""
echo_success "Data generation complete!"
echo ""
echo_info "Next steps:"
echo "  1. Trigger rule-updater to rebuild Redis snapshot:"
echo "     curl -X POST https://your-api/api/v1/rules (create/update any rule)"
echo ""
echo "  2. Or restart rule-updater service to force snapshot rebuild"
