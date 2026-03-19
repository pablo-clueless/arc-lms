-- Drop tenants table
DROP INDEX IF EXISTS idx_tenants_created_at;
DROP INDEX IF EXISTS idx_tenants_principal_admin_id;
DROP INDEX IF EXISTS idx_tenants_status;
DROP TABLE IF EXISTS tenants CASCADE;
