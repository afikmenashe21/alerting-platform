-- Revert wildcard support
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

-- Restore original constraint
ALTER TABLE rules ADD CONSTRAINT rules_severity_check 
    CHECK (severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL'));
