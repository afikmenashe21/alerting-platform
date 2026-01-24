-- Generate Test Data for Alerting Platform
-- Target: 1,500 clients, 450,000 rules, 900,000 endpoints

-- Clean existing data
DELETE FROM endpoints;
DELETE FROM rules;
DELETE FROM notifications;
DELETE FROM clients;

-- Create clients
INSERT INTO clients (client_id, name, created_at, updated_at)
SELECT 
    'client-' || LPAD(i::text, 5, '0'),
    'Client ' || i,
    NOW(),
    NOW()
FROM generate_series(1, 1500) AS i;

-- Create rules (300 per client = 450,000 total)
-- Using 300 combinations of (severity, source, name)
INSERT INTO rules (client_id, severity, source, name, enabled, version, created_at, updated_at)
SELECT 
    c.client_id,
    sev.severity,
    src.source,
    n.name,
    TRUE,
    1,
    NOW(),
    NOW()
FROM clients c
CROSS JOIN (VALUES ('LOW'), ('MEDIUM'), ('HIGH'), ('CRITICAL')) AS sev(severity)
CROSS JOIN (VALUES ('api'), ('db'), ('cache'), ('monitor'), ('queue'), ('worker'), ('frontend'), ('backend')) AS src(source)
CROSS JOIN (VALUES ('timeout'), ('error'), ('crash'), ('slow'), ('memory'), ('cpu'), ('disk'), ('network'), ('auth')) AS n(name)
CROSS JOIN generate_series(1, 1) AS dummy;  -- 4 severities * 8 sources * 9 names = 288 rules per client (close to 300)

-- Update to add 'validation' name to some combinations to reach ~300 per client
INSERT INTO rules (client_id, severity, source, name, enabled, version, created_at, updated_at)
SELECT 
    c.client_id,
    sev.severity,
    src.source,
    'validation',
    TRUE,
    1,
    NOW(),
    NOW()
FROM clients c
CROSS JOIN (VALUES ('LOW'), ('MEDIUM'), ('HIGH')) AS sev(severity)  -- Only 3 severities for validation
CROSS JOIN (VALUES ('api'), ('db'), ('cache'), ('monitor')) AS src(source)  -- Only 4 sources
ON CONFLICT (client_id, severity, source, name) DO NOTHING;

-- Create endpoints (2 per rule = 900,000 total)
-- Email endpoints
INSERT INTO endpoints (rule_id, type, value, enabled, created_at, updated_at)
SELECT 
    r.rule_id,
    'email',
    'alert-' || ROW_NUMBER() OVER () || '@example.com',
    TRUE,
    NOW(),
    NOW()
FROM rules r;

-- Webhook endpoints
INSERT INTO endpoints (rule_id, type, value, enabled, created_at, updated_at)
SELECT 
    r.rule_id,
    'webhook',
    'https://webhook.example.com/rule/' || LEFT(r.rule_id::text, 8),
    TRUE,
    NOW(),
    NOW()
FROM rules r;

-- Show summary
SELECT 'Clients' as entity, COUNT(*) as count FROM clients
UNION ALL
SELECT 'Rules', COUNT(*) FROM rules
UNION ALL
SELECT 'Endpoints', COUNT(*) FROM endpoints;
