-- Allow wildcard "*" in rules table
-- This enables rules to match any value for a field (e.g., */test-source/test-name matches any severity)

-- Drop existing severity CHECK constraint if it exists (may have auto-generated name)
DO $$
DECLARE
    constraint_name text;
BEGIN
    -- Find the constraint that checks severity IN (...)
    SELECT conname INTO constraint_name
    FROM pg_constraint
    WHERE conrelid = 'rules'::regclass
      AND contype = 'c'
      AND (pg_get_constraintdef(oid) LIKE '%severity%IN%' OR conname LIKE '%severity%');
    
    -- Drop it if found
    IF constraint_name IS NOT NULL THEN
        EXECUTE format('ALTER TABLE rules DROP CONSTRAINT IF EXISTS %I', constraint_name);
    END IF;
END $$;

-- Add new constraint allowing "*" as wildcard
ALTER TABLE rules ADD CONSTRAINT rules_severity_check 
    CHECK (severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL', '*'));

-- Note: The unique constraint (client_id, severity, source, name) still applies
-- This means you can have multiple rules with wildcards, but not exact duplicates
-- Source and name fields can already accept "*" (no CHECK constraint on them)
