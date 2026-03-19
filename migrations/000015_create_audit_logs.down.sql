-- Drop audit_logs table
DROP INDEX IF EXISTS idx_audit_logs_tenant_id_actor_user_id;
DROP INDEX IF EXISTS idx_audit_logs_tenant_id_resource_type;
DROP INDEX IF EXISTS idx_audit_logs_tenant_id_timestamp;
DROP INDEX IF EXISTS idx_audit_logs_timestamp;
DROP INDEX IF EXISTS idx_audit_logs_is_sensitive;
DROP INDEX IF EXISTS idx_audit_logs_resource_id;
DROP INDEX IF EXISTS idx_audit_logs_resource_type;
DROP INDEX IF EXISTS idx_audit_logs_action;
DROP INDEX IF EXISTS idx_audit_logs_actor_user_id;
DROP INDEX IF EXISTS idx_audit_logs_tenant_id;
DROP TABLE IF EXISTS audit_logs CASCADE;
