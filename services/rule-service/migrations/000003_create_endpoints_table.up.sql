-- Create endpoints table for control-plane
-- Endpoints represent notification destinations for rules
-- A rule can have multiple endpoints (e.g., email, webhook, Slack)

CREATE TABLE IF NOT EXISTS endpoints (
    endpoint_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID NOT NULL REFERENCES rules(rule_id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL CHECK (type IN ('email', 'webhook', 'slack')), -- Endpoint type
    value VARCHAR(500) NOT NULL, -- Endpoint value (email address, URL, etc.)
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Ensure unique endpoint per rule with same type and value
    CONSTRAINT endpoints_rule_type_value_unique UNIQUE (rule_id, type, value)
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_endpoints_rule_id ON endpoints(rule_id);
CREATE INDEX IF NOT EXISTS idx_endpoints_enabled ON endpoints(enabled) WHERE enabled = TRUE;
CREATE INDEX IF NOT EXISTS idx_endpoints_type ON endpoints(type);
