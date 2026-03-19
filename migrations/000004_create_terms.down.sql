-- Drop terms tables
DROP INDEX IF EXISTS idx_term_date_ranges_tenant_id;
DROP INDEX IF EXISTS idx_term_date_ranges_term_id;
DROP INDEX IF EXISTS idx_terms_end_date;
DROP INDEX IF EXISTS idx_terms_start_date;
DROP INDEX IF EXISTS idx_terms_tenant_id_status;
DROP INDEX IF EXISTS idx_terms_status;
DROP INDEX IF EXISTS idx_terms_session_id;
DROP INDEX IF EXISTS idx_terms_tenant_id_id;
DROP INDEX IF EXISTS idx_terms_tenant_id;
DROP TABLE IF EXISTS term_date_ranges CASCADE;
DROP TABLE IF EXISTS terms CASCADE;
