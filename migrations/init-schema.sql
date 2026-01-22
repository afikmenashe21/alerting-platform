-- Complete database schema initialization
-- Run this once to set up all tables for the alerting platform

-- Drop existing tables to recreate with correct schema
DROP TABLE IF EXISTS endpoints CASCADE;
DROP TABLE IF EXISTS notifications CASCADE;
DROP TABLE IF EXISTS rules CASCADE;
DROP TABLE IF EXISTS clients CASCADE;

-- Create clients table
CREATE TABLE clients (
    client_id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create rules table
CREATE TABLE rules (
    rule_id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    client_id VARCHAR(255) NOT NULL REFERENCES clients(client_id) ON DELETE CASCADE,
    severity VARCHAR(50),
    source VARCHAR(255),
    name VARCHAR(255),
    enabled BOOLEAN DEFAULT TRUE,
    version INTEGER DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(client_id, severity, source, name)
);

-- Create endpoints table (linked to rules, not clients)
CREATE TABLE endpoints (
    endpoint_id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    rule_id VARCHAR(255) NOT NULL REFERENCES rules(rule_id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    value TEXT NOT NULL,
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(rule_id, type, value)
);

-- Create notifications table
CREATE TABLE notifications (
    notification_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id VARCHAR(255) NOT NULL,
    alert_id VARCHAR(255) NOT NULL,
    severity VARCHAR(50),
    source VARCHAR(255),
    name VARCHAR(255),
    context JSONB,
    rule_ids TEXT[],
    status VARCHAR(50) DEFAULT 'RECEIVED',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(client_id, alert_id)
);

-- Indexes
CREATE INDEX idx_rules_enabled ON rules(enabled) WHERE enabled = TRUE;
CREATE INDEX idx_rules_client ON rules(client_id);
CREATE INDEX idx_endpoints_rule ON endpoints(rule_id);
CREATE INDEX idx_notifications_client_status ON notifications(client_id, status);
CREATE INDEX idx_notifications_created_at ON notifications(created_at);

-- Verify tables were created
SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename;
