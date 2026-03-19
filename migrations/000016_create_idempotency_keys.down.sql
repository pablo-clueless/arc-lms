-- Drop idempotency_keys table
DROP INDEX IF EXISTS idx_idempotency_keys_created_at;
DROP INDEX IF EXISTS idx_idempotency_keys_expires_at;
DROP INDEX IF EXISTS idx_idempotency_keys_user_id;
DROP INDEX IF EXISTS idx_idempotency_keys_tenant_id;
DROP INDEX IF EXISTS idx_idempotency_keys_key;
DROP TABLE IF EXISTS idempotency_keys CASCADE;
