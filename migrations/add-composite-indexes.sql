-- Composite indexes for filtering + ordering performance
-- Run this on existing databases to optimize filtered paginated queries
-- These indexes are safe to add on production (CONCURRENTLY prevents table locks)

-- Composite index for rules filtered by client_id and ordered by created_at
-- Optimizes: SELECT * FROM rules WHERE client_id = $1 ORDER BY created_at DESC
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_rules_client_created_at
ON rules(client_id, created_at DESC);

-- Composite index for endpoints filtered by rule_id and ordered by created_at
-- Optimizes: SELECT * FROM endpoints WHERE rule_id = $1 ORDER BY created_at DESC
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_endpoints_rule_created_at
ON endpoints(rule_id, created_at DESC);

-- Composite index for notifications filtered by client_id ordered by created_at
-- Optimizes: SELECT * FROM notifications WHERE client_id = $1 ORDER BY created_at DESC
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_notifications_client_created_at
ON notifications(client_id, created_at DESC);

-- Composite index for notifications filtered by status ordered by created_at
-- Optimizes: SELECT * FROM notifications WHERE status = $1 ORDER BY created_at DESC
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_notifications_status_created_at
ON notifications(status, created_at DESC);

-- Analyze tables to update statistics for query planner
ANALYZE clients;
ANALYZE rules;
ANALYZE endpoints;
ANALYZE notifications;

-- Verify indexes were created
SELECT indexname, tablename
FROM pg_indexes
WHERE schemaname = 'public'
AND indexname LIKE '%created_at'
ORDER BY tablename, indexname;
