-- Counts cache table for exact counts without full table scans
-- This table stores exact counts that get refreshed periodically
-- Much faster than COUNT(*) on large tables

-- Create counts cache table
CREATE TABLE IF NOT EXISTS table_counts (
    table_name VARCHAR(50) PRIMARY KEY,
    row_count BIGINT NOT NULL DEFAULT 0,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Initialize with current counts
INSERT INTO table_counts (table_name, row_count, last_updated)
VALUES
    ('clients', (SELECT COUNT(*) FROM clients), NOW()),
    ('rules', (SELECT COUNT(*) FROM rules), NOW()),
    ('endpoints', (SELECT COUNT(*) FROM endpoints), NOW()),
    ('notifications', (SELECT COUNT(*) FROM notifications), NOW())
ON CONFLICT (table_name) DO UPDATE SET
    row_count = EXCLUDED.row_count,
    last_updated = NOW();

-- Function to refresh a specific count
CREATE OR REPLACE FUNCTION refresh_table_count(p_table_name VARCHAR)
RETURNS BIGINT AS $$
DECLARE
    v_count BIGINT;
BEGIN
    EXECUTE format('SELECT COUNT(*) FROM %I', p_table_name) INTO v_count;

    INSERT INTO table_counts (table_name, row_count, last_updated)
    VALUES (p_table_name, v_count, NOW())
    ON CONFLICT (table_name) DO UPDATE SET
        row_count = v_count,
        last_updated = NOW();

    RETURN v_count;
END;
$$ LANGUAGE plpgsql;

-- Function to refresh all counts
CREATE OR REPLACE FUNCTION refresh_all_counts()
RETURNS void AS $$
BEGIN
    PERFORM refresh_table_count('clients');
    PERFORM refresh_table_count('rules');
    PERFORM refresh_table_count('endpoints');
    PERFORM refresh_table_count('notifications');
END;
$$ LANGUAGE plpgsql;

-- Function to get cached count (with optional staleness threshold)
-- If cache is older than threshold_seconds, returns approximate count from pg_stat
CREATE OR REPLACE FUNCTION get_cached_count(
    p_table_name VARCHAR,
    p_max_age_seconds INTEGER DEFAULT 300  -- 5 minutes default
)
RETURNS BIGINT AS $$
DECLARE
    v_count BIGINT;
    v_last_updated TIMESTAMP;
BEGIN
    SELECT row_count, last_updated INTO v_count, v_last_updated
    FROM table_counts
    WHERE table_name = p_table_name;

    IF v_count IS NULL OR v_last_updated < NOW() - (p_max_age_seconds || ' seconds')::INTERVAL THEN
        -- Cache miss or stale - refresh
        v_count := refresh_table_count(p_table_name);
    END IF;

    RETURN v_count;
END;
$$ LANGUAGE plpgsql;

-- Triggers to keep counts approximately up-to-date
-- These increment/decrement counts on INSERT/DELETE
-- Note: For high-throughput tables, consider using a background job instead

-- Clients trigger
CREATE OR REPLACE FUNCTION update_clients_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE table_counts SET row_count = row_count + 1 WHERE table_name = 'clients';
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE table_counts SET row_count = row_count - 1 WHERE table_name = 'clients';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_clients_count ON clients;
CREATE TRIGGER trg_clients_count
    AFTER INSERT OR DELETE ON clients
    FOR EACH ROW EXECUTE FUNCTION update_clients_count();

-- Rules trigger
CREATE OR REPLACE FUNCTION update_rules_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE table_counts SET row_count = row_count + 1 WHERE table_name = 'rules';
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE table_counts SET row_count = row_count - 1 WHERE table_name = 'rules';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_rules_count ON rules;
CREATE TRIGGER trg_rules_count
    AFTER INSERT OR DELETE ON rules
    FOR EACH ROW EXECUTE FUNCTION update_rules_count();

-- Endpoints trigger
CREATE OR REPLACE FUNCTION update_endpoints_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE table_counts SET row_count = row_count + 1 WHERE table_name = 'endpoints';
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE table_counts SET row_count = row_count - 1 WHERE table_name = 'endpoints';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_endpoints_count ON endpoints;
CREATE TRIGGER trg_endpoints_count
    AFTER INSERT OR DELETE ON endpoints
    FOR EACH ROW EXECUTE FUNCTION update_endpoints_count();

-- Notifications trigger
CREATE OR REPLACE FUNCTION update_notifications_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE table_counts SET row_count = row_count + 1 WHERE table_name = 'notifications';
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE table_counts SET row_count = row_count - 1 WHERE table_name = 'notifications';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_notifications_count ON notifications;
CREATE TRIGGER trg_notifications_count
    AFTER INSERT OR DELETE ON notifications
    FOR EACH ROW EXECUTE FUNCTION update_notifications_count();

-- Verify setup
SELECT 'Counts cache setup complete' as status;
SELECT * FROM table_counts;
