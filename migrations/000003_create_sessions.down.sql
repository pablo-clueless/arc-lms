-- Drop sessions table
DROP INDEX IF EXISTS idx_sessions_tenant_id_status;
DROP INDEX IF EXISTS idx_sessions_status;
DROP INDEX IF EXISTS idx_sessions_tenant_id_id;
DROP INDEX IF EXISTS idx_sessions_tenant_id;
DROP INDEX IF EXISTS idx_sessions_one_active_per_tenant;
DROP TABLE IF EXISTS sessions CASCADE;
