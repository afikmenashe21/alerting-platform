-- Remove email field from rules table
-- Email endpoints are now managed through the endpoints table

ALTER TABLE rules DROP COLUMN IF EXISTS email;
