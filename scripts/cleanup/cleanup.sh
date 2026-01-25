#!/bin/bash
set -e

echo "========================================"
echo "  Full System Reset"
echo "========================================"
echo ""

# Step 1: Redis cleanup
echo "Step 1: Clearing Redis cache..."
echo "  Connecting to $REDIS_HOST:${REDIS_PORT:-6379}"

# Delete ALL keys in Redis (full flush for clean reset)
redis-cli -h "$REDIS_HOST" -p "${REDIS_PORT:-6379}" FLUSHALL
echo "  ✓ Redis FLUSHALL complete - all keys deleted"

# Show remaining keys
REMAINING=$(redis-cli -h "$REDIS_HOST" -p "${REDIS_PORT:-6379}" DBSIZE)
echo "  $REMAINING"

echo ""
echo "Step 2: Cleaning database notifications..."
echo "  Connecting to $DB_HOST:${DB_PORT:-5432}"

PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "${DB_PORT:-5432}" -U "${DB_USER:-postgres}" -d "${DB_NAME:-alerting}" << 'EOSQL'
-- Truncate notifications
TRUNCATE TABLE notifications;

-- Reset count in cache
UPDATE table_counts SET row_count = 0, last_updated = NOW() WHERE table_name = 'notifications';

-- Show final counts
SELECT '✓ Notifications truncated' as status;
SELECT table_name, row_count FROM table_counts ORDER BY table_name;
EOSQL

echo ""
echo "========================================"
echo "  ✓ Full System Reset Complete!"
echo "========================================"
