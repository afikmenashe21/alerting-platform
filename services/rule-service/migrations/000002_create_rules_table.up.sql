-- Create rules table for control-plane
-- Rules define alert matching criteria and notification endpoints

CREATE TABLE IF NOT EXISTS rules (
    rule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id VARCHAR(255) NOT NULL REFERENCES clients(client_id) ON DELETE CASCADE,
    severity VARCHAR(50) NOT NULL CHECK (severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    source VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255), -- Override email endpoint for this rule (optional, falls back to client email)
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    version INTEGER NOT NULL DEFAULT 1, -- Optimistic locking version
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Ensure unique rule per client with same criteria
    CONSTRAINT rules_client_criteria_unique UNIQUE (client_id, severity, source, name)
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_rules_client_id ON rules(client_id);
CREATE INDEX IF NOT EXISTS idx_rules_enabled ON rules(enabled) WHERE enabled = TRUE;
CREATE INDEX IF NOT EXISTS idx_rules_updated_at ON rules(updated_at);
CREATE INDEX IF NOT EXISTS idx_rules_severity_source_name ON rules(severity, source, name);
