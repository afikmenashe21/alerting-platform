-- Delete all notifications
TRUNCATE TABLE notifications;

-- Refresh counts cache
UPDATE table_counts SET row_count = 0, last_updated = NOW() WHERE table_name = 'notifications';

-- Verify
SELECT 'Notifications deleted' as status;
SELECT * FROM table_counts;
