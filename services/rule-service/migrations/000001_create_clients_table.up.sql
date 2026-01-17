-- Create clients table for control-plane
-- Clients represent tenants/organizations that define alerting rules

CREATE TABLE IF NOT EXISTS clients (
    client_id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255), -- Default email endpoint for notifications
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Index for lookups
CREATE INDEX IF NOT EXISTS idx_clients_name ON clients(name);
