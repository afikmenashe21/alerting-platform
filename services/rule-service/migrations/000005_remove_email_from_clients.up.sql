-- Remove email field from clients table
-- Email endpoints are now managed through the endpoints table on rules

ALTER TABLE clients DROP COLUMN IF EXISTS email;
