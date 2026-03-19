-- Drop users table
DROP INDEX IF EXISTS idx_users_invitation_token;
DROP INDEX IF EXISTS idx_users_password_reset_token;
DROP INDEX IF EXISTS idx_users_tenant_id_role;
DROP INDEX IF EXISTS idx_users_status;
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_tenant_id_id;
DROP INDEX IF EXISTS idx_users_tenant_id;
DROP TABLE IF EXISTS users CASCADE;
