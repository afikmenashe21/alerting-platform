-- Balanced Test Data Generator for Alerting Platform
-- Target: Smaller dataset that won't overwhelm free-tier resources
--
-- Configuration:
--   - 200 clients
--   - 4 rules per client = 800 rules total
--   - 2 endpoints per rule = 1,600 endpoints total
--   - Even distribution across severities/sources
--
-- This cleans ALL data including notifications!

-- ============================================
-- Step 1: Clean ALL existing data
-- ============================================
TRUNCATE TABLE endpoints CASCADE;
TRUNCATE TABLE rules CASCADE;
TRUNCATE TABLE notifications CASCADE;
TRUNCATE TABLE clients CASCADE;

-- Reset sequences if any
SELECT setval(pg_get_serial_sequence('notifications', 'notification_id'), 1, false)
WHERE pg_get_serial_sequence('notifications', 'notification_id') IS NOT NULL;

-- ============================================
-- Step 2: Create 200 clients
-- ============================================
INSERT INTO clients (client_id, name, created_at, updated_at)
SELECT
    'client-' || LPAD(i::text, 5, '0'),
    'Client ' || i,
    NOW() - (RANDOM() * INTERVAL '30 days'),  -- Spread creation dates
    NOW()
FROM generate_series(1, 200) AS i;

-- ============================================
-- Step 3: Create 4 rules per client (800 total)
-- Using varied combinations of severity/source/name
-- ============================================

-- Rule 1: LOW severity, api source
INSERT INTO rules (client_id, severity, source, name, enabled, version, created_at, updated_at)
SELECT
    c.client_id,
    'LOW',
    'api',
    'timeout',
    TRUE,
    1,
    NOW() - (RANDOM() * INTERVAL '30 days'),
    NOW()
FROM clients c;

-- Rule 2: MEDIUM severity, db source
INSERT INTO rules (client_id, severity, source, name, enabled, version, created_at, updated_at)
SELECT
    c.client_id,
    'MEDIUM',
    'db',
    'error',
    TRUE,
    1,
    NOW() - (RANDOM() * INTERVAL '30 days'),
    NOW()
FROM clients c;

-- Rule 3: HIGH severity, cache source
INSERT INTO rules (client_id, severity, source, name, enabled, version, created_at, updated_at)
SELECT
    c.client_id,
    'HIGH',
    'cache',
    'slow',
    TRUE,
    1,
    NOW() - (RANDOM() * INTERVAL '30 days'),
    NOW()
FROM clients c;

-- Rule 4: CRITICAL severity, monitor source
INSERT INTO rules (client_id, severity, source, name, enabled, version, created_at, updated_at)
SELECT
    c.client_id,
    'CRITICAL',
    'monitor',
    'crash',
    TRUE,
    1,
    NOW() - (RANDOM() * INTERVAL '30 days'),
    NOW()
FROM clients c;

-- ============================================
-- Step 4: Create 2 endpoints per rule (1,600 total)
-- ============================================

-- Email endpoints for all rules
INSERT INTO endpoints (rule_id, type, value, enabled, created_at, updated_at)
SELECT
    r.rule_id,
    'email',
    'alert-' || ROW_NUMBER() OVER (ORDER BY r.created_at) || '@example.com',
    TRUE,
    NOW() - (RANDOM() * INTERVAL '30 days'),
    NOW()
FROM rules r;

-- Webhook endpoints for all rules
INSERT INTO endpoints (rule_id, type, value, enabled, created_at, updated_at)
SELECT
    r.rule_id,
    'webhook',
    'https://webhook.example.com/rule/' || LEFT(r.rule_id::text, 8),
    TRUE,
    NOW() - (RANDOM() * INTERVAL '30 days'),
    NOW()
FROM rules r;

-- ============================================
-- Step 5: Update pg_stat statistics for accurate counts
-- ============================================
ANALYZE clients;
ANALYZE rules;
ANALYZE endpoints;
ANALYZE notifications;

-- ============================================
-- Summary
-- ============================================
SELECT 'Summary' as info;
SELECT '========' as info;
SELECT 'Clients:       ' || COUNT(*) FROM clients;
SELECT 'Rules:         ' || COUNT(*) FROM rules;
SELECT 'Endpoints:     ' || COUNT(*) FROM endpoints;
SELECT 'Notifications: ' || COUNT(*) FROM notifications;

-- Verify distribution
SELECT 'Rules per client:' as info;
SELECT client_id, COUNT(*) as rules
FROM rules
GROUP BY client_id
ORDER BY client_id
LIMIT 5;

SELECT 'Endpoints per rule:' as info;
SELECT r.rule_id, COUNT(e.endpoint_id) as endpoints
FROM rules r
LEFT JOIN endpoints e ON r.rule_id = e.rule_id
GROUP BY r.rule_id
LIMIT 5;
