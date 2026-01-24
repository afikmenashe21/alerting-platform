-- Remove index on created_at for notifications table
DROP INDEX IF EXISTS idx_notifications_created_at;
