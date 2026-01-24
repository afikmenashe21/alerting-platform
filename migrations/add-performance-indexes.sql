-- Performance indexes migration
-- Run this on existing databases to add missing indexes for pagination performance
-- These indexes are safe to add on production (CONCURRENTLY option prevents table locks)

-- Add index for notifications status GROUP BY (dashboard)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_notifications_status ON notifications(status);

-- Add index for rules created_at ORDER BY (pagination)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_rules_created_at ON rules(created_at);

-- Add index for endpoints created_at ORDER BY (pagination)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_endpoints_created_at ON endpoints(created_at);

-- Add index for clients created_at ORDER BY (pagination)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_clients_created_at ON clients(created_at);

-- Verify indexes were created
SELECT indexname, tablename FROM pg_indexes 
WHERE schemaname = 'public' 
ORDER BY tablename, indexname;
