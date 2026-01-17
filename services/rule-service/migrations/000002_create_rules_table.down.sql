-- Drop rules table

DROP INDEX IF EXISTS idx_rules_severity_source_name;
DROP INDEX IF EXISTS idx_rules_updated_at;
DROP INDEX IF EXISTS idx_rules_enabled;
DROP INDEX IF EXISTS idx_rules_client_id;
DROP TABLE IF EXISTS rules;
