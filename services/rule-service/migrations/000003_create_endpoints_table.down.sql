-- Drop endpoints table

DROP INDEX IF EXISTS idx_endpoints_type;
DROP INDEX IF EXISTS idx_endpoints_enabled;
DROP INDEX IF EXISTS idx_endpoints_rule_id;
DROP TABLE IF EXISTS endpoints;
