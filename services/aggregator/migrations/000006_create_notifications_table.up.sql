-- Create notifications table with idempotency boundary
-- Unique constraint on (client_id, alert_id) ensures no duplicates per client per alert
-- 
-- Migration: 000006
-- Service: aggregator
-- Depends on: rule-service migrations 000001-000005 (clients table must exist)
-- See: ../migrations/MIGRATION_STRATEGY.md for versioning strategy

CREATE TABLE IF NOT EXISTS notifications (
    notification_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id VARCHAR(255) NOT NULL,
    alert_id VARCHAR(255) NOT NULL,
    severity VARCHAR(50) NOT NULL,
    source VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    context JSONB,
    rule_ids TEXT[] NOT NULL, -- Array of rule IDs that matched
    status VARCHAR(50) NOT NULL DEFAULT 'RECEIVED',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Idempotency boundary: one notification per client per alert
    CONSTRAINT notifications_client_alert_unique UNIQUE (client_id, alert_id)
);

-- Index for status lookups (used by sender service)
CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status) WHERE status = 'RECEIVED';

-- Index for client lookups
CREATE INDEX IF NOT EXISTS idx_notifications_client_id ON notifications(client_id);

-- Index for alert lookups
CREATE INDEX IF NOT EXISTS idx_notifications_alert_id ON notifications(alert_id);
