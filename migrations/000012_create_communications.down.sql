-- Drop emails table
DROP INDEX IF EXISTS idx_emails_created_at;
DROP INDEX IF EXISTS idx_emails_scheduled_for;
DROP INDEX IF EXISTS idx_emails_status;
DROP INDEX IF EXISTS idx_emails_sender_id;
DROP INDEX IF EXISTS idx_emails_tenant_id_id;
DROP INDEX IF EXISTS idx_emails_tenant_id;
DROP TABLE IF EXISTS emails CASCADE;
