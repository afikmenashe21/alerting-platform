#!/bin/sh
set -e

echo "================================"
echo "Database Migration Runner"
echo "================================"
echo ""

if [ -z "$DB_HOST" ] || [ -z "$DB_PASSWORD" ]; then
    echo "ERROR: DB_HOST and DB_PASSWORD must be set"
    exit 1
fi

DB_NAME="${DB_NAME:-alerting}"
DB_USER="${DB_USER:-postgres}"
DB_PORT="${DB_PORT:-5432}"

echo "Connecting to: $DB_HOST:$DB_PORT/$DB_NAME"
echo "User: $DB_USER"
echo ""

# Wait for database to be ready
echo "Waiting for database to be ready..."
until PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c '\q' 2>/dev/null; do
    echo "  Database not ready, waiting..."
    sleep 2
done

echo "✓ Database is ready"
echo ""

# Run migration
echo "Running migrations..."
echo "================================"
PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" < /migrations/init-schema.sql

echo ""
echo "================================"
echo "Verifying schema..."
echo "================================"
PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "
SELECT 
    'Table: ' || tablename as info
FROM pg_tables 
WHERE schemaname = 'public'
ORDER BY tablename;
"

echo ""
echo "✓ Migration completed successfully!"
