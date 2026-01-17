-- Drop notifications table and indexes

DROP INDEX IF EXISTS idx_notifications_alert_id;
DROP INDEX IF EXISTS idx_notifications_client_id;
DROP INDEX IF EXISTS idx_notifications_status;
DROP TABLE IF EXISTS notifications;
