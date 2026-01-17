-- Restore email field to rules table (for rollback)

ALTER TABLE rules ADD COLUMN email VARCHAR(255);
