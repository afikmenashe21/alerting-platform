-- Add index on created_at for notifications table
-- Required for metrics-service queries that filter by time range
-- 
-- Migration: 000007
-- Service: aggregator

-- Note: Using regular CREATE INDEX instead of CONCURRENTLY because migrations run in transactions
-- For very large tables, consider running CONCURRENTLY manually outside the migration
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at);
