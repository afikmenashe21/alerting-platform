-- Restore email field to clients table (for rollback)

ALTER TABLE clients ADD COLUMN email VARCHAR(255);
