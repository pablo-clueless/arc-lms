-- Drop notifications table
DROP INDEX IF EXISTS idx_notifications_created_at;
DROP INDEX IF EXISTS idx_notifications_tenant_id_user_id_unread;
DROP INDEX IF EXISTS idx_notifications_tenant_id_user_id;
DROP INDEX IF EXISTS idx_notifications_priority;
DROP INDEX IF EXISTS idx_notifications_is_read;
DROP INDEX IF EXISTS idx_notifications_event_type;
DROP INDEX IF EXISTS idx_notifications_user_id;
DROP INDEX IF EXISTS idx_notifications_tenant_id_id;
DROP INDEX IF EXISTS idx_notifications_tenant_id;
DROP TABLE IF EXISTS notifications CASCADE;
