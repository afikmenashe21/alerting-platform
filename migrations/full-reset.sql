-- Full system reset: Delete notifications, reset counts

-- Delete all notifications
TRUNCATE TABLE notifications;

-- Reset notification count in cache
UPDATE table_counts SET row_count = 0, last_updated = NOW() WHERE table_name = 'notifications';

-- Verify counts
SELECT 'Reset complete' as status;
SELECT * FROM table_counts ORDER BY table_name;
