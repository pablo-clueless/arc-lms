-- Drop classes table
DROP INDEX IF EXISTS idx_classes_level;
DROP INDEX IF EXISTS idx_classes_tenant_id_session_id;
DROP INDEX IF EXISTS idx_classes_status;
DROP INDEX IF EXISTS idx_classes_session_id;
DROP INDEX IF EXISTS idx_classes_tenant_id_id;
DROP INDEX IF EXISTS idx_classes_tenant_id;
DROP TABLE IF EXISTS classes CASCADE;
