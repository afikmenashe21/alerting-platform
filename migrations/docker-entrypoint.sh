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

# Check if tables exist (to determine if this is a new or existing database)
TABLE_COUNT=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM pg_tables WHERE schemaname = 'public' AND tablename IN ('clients', 'rules', 'endpoints', 'notifications');" | tr -d ' ')

if [ "$TABLE_COUNT" = "4" ]; then
    echo "Existing database detected - running incremental migrations only"
    echo "================================"

    # Run only the index migrations (safe for existing data)
    echo "Adding composite indexes for performance..."
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" < /migrations/add-composite-indexes.sql
else
    echo "New database detected - running full schema initialization"
    echo "================================"

    # Run full init for new databases
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" < /migrations/init-schema.sql
fi

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
echo "Indexes:"
PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "
SELECT indexname, tablename
FROM pg_indexes
WHERE schemaname = 'public'
ORDER BY tablename, indexname;
"

echo ""
echo "✓ Migration completed successfully!"
